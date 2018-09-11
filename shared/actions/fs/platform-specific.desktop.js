// @flow
import * as FsGen from '../fs-gen'
import * as Saga from '../../util/saga'
import {downloadFolder} from '../../util/file.desktop'
import * as Config from '../../constants/config'
import * as RPCTypes from '../../constants/types/rpc-gen'
import * as Types from '../../constants/types/fs'
import * as Constants from '../../constants/fs'
import * as SafeElectron from '../../util/safe-electron.desktop'
import fs from 'fs'
import type {TypedState} from '../../constants/reducer'
import {fileUIName, isLinux, isWindows} from '../../constants/platform'
import {fsTab} from '../../constants/tabs'
import logger from '../../logger'
import {spawn, execFileSync, execSync} from 'child_process'
import path from 'path'
import {navigateTo} from '../route-tree'

type pathType = 'file' | 'directory'

// pathToURL takes path and converts to (file://) url.
// See https://github.com/sindresorhus/file-url
function pathToURL(path: string): string {
  let goodPath = path.replace(/\\/g, '/')

  // Windows drive letter must be prefixed with a slash
  if (goodPath[0] !== '/') {
    goodPath = '/' + goodPath
  }

  return encodeURI('file://' + goodPath).replace(/#/g, '%23')
}

function openInDefaultDirectory(openPath: string) {
  return new Promise((resolve, reject) => {
    // Paths in directories might be symlinks, so resolve using
    // realpath.
    // For example /keybase/private/gabrielh,chris gets redirected to
    // /keybase/private/chris,gabrielh.
    fs.realpath(openPath, (err, resolvedPath) => {
      if (err) {
        reject(new Error(`No realpath for ${openPath}: ${err}`))
        return
      }
      // Convert to URL for openExternal call.
      // We use openExternal instead of openItem because it
      // correctly focuses' the Finder, and also uses a newer
      // native API on macOS.
      const url = pathToURL(resolvedPath)
      logger.info('Open URL (directory):', url)

      SafeElectron.getShell().openExternal(url, {activate: true}, err => {
        if (err) {
          reject(err)
          return
        }
        logger.info('Opened directory:', openPath)
        resolve()
      })
    })
  })
}

function getPathType(openPath: string): Promise<pathType> {
  return new Promise((resolve, reject) => {
    fs.stat(openPath, (err, stats) => {
      if (err) {
        reject(new Error(`Unable to open/stat file: ${openPath}`))
        return
      }
      if (stats.isFile()) {
        resolve('file')
      } else if (stats.isDirectory()) {
        resolve('directory')
      } else {
        reject(new Error(`Unable to open: Not a file or directory`))
      }
    })
  })
}

function _open(openPath: string) {
  return new Promise((resolve, reject) => {
    getPathType(openPath).then(typ => {
      if (typ === 'directory') {
        if (isWindows) {
          if (!SafeElectron.getShell().openItem(openPath)) {
            reject(new Error(`Unable to open item: ${openPath}`))
            return
          }
        } else {
          openInDefaultDirectory(openPath).then(resolve, reject)
          return
        }
      } else if (typ === 'file') {
        if (!SafeElectron.getShell().showItemInFolder(openPath)) {
          reject(new Error(`Unable to open item in folder: ${openPath}`))
          return
        }
      } else {
        reject(new Error(`Invalid path type`))
        return
      }
      resolve()
    })
  })
}

function* openWithCurrentMountDir(openPath: string): Saga.SagaGenerator<any, any> {
  const goodPath = path.posix.normalize(openPath)
  if (!openPath.startsWith(Config.defaultKBFSPath)) {
    throw new Error(`openWithCurrentMountDir requires ${Config.defaultKBFSPath} prefix: ${openPath}`)
  }

  // turns '/keybase/private/alice' to 'private/alice'
  const subPath = goodPath
    .split('/')
    .slice(2)
    .join(path.sep)

  const kbfsPath = yield Saga.call(RPCTypes.kbfsMountGetCurrentMountDirRpcPromise)

  if (!kbfsPath) {
    throw new Error('No kbfsPath (RPC)')
  }

  const resolvedPath = path.resolve(kbfsPath, subPath)
  // Check to make sure our resolved path starts with the kbfsPath
  // i.e. (not opening a folder outside kbfs)
  if (!resolvedPath.startsWith(kbfsPath)) {
    throw new Error(`openWithCurrentMountDir requires ${kbfsPath} prefix: ${goodPath}`)
  }

  yield Saga.call(_open, resolvedPath)
}

function openInFileUISaga({payload: {path}}: FsGen.OpenInFileUIPayload, state: TypedState) {
  const openPath = path || downloadFolder
  const enabled = state.fs.fuseStatus && state.fs.fuseStatus.kextStarted
  if (isLinux || enabled) {
    return Saga.call(openPath.startsWith(Config.defaultKBFSPath) ? openWithCurrentMountDir : _open, openPath)
  } else {
    return Saga.put(navigateTo([fsTab, {props: {path: Types.stringToPath(openPath)}, selected: 'folder'}]))
  }
}

function waitForMount(attempt: number) {
  return new Promise((resolve, reject) => {
    // Read the KBFS path waiting for files to exist, which means it's mounted
    // TODO: should handle current mount directory
    fs.readdir(`${Config.defaultKBFSPath}${Config.defaultPrivatePrefix}`, (err, files) => {
      if (!err && files.length > 0) {
        resolve(true)
      } else if (attempt > 15) {
        reject(new Error(`${Config.defaultKBFSPath} is unavailable. Please try again.`))
      } else {
        setTimeout(() => {
          waitForMount(attempt + 1).then(resolve, reject)
        }, 1000)
      }
    })
  })
}

const installKBFSSuccess = (result: RPCTypes.InstallResult) =>
  Saga.sequentially([
    Saga.call(waitForMount, 0),
    Saga.put(FsGen.createSetFlags({kbfsInstalling: false, showBanner: true})),
  ])

function fuseStatusResultSaga({payload: {prevStatus, status}}: FsGen.FuseStatusResultPayload) {
  // If our kextStarted status changed, finish KBFS install
  if (status.kextStarted && prevStatus && !prevStatus.kextStarted) {
    return Saga.put(FsGen.createInstallKBFS())
  }
}

function* fuseStatusSaga(): Saga.SagaGenerator<any, any> {
  const state: TypedState = yield Saga.select()
  const prevStatus = state.fs.fuseStatus

  let status = yield Saga.call(RPCTypes.installFuseStatusRpcPromise, {bundleVersion: ''})
  if (isWindows && status.installStatus !== RPCTypes.installInstallStatus.installed) {
    // Check if another Dokan we didn't install mounted the filesystem
    const kbfsMount = yield Saga.call(RPCTypes.kbfsMountGetCurrentMountDirRpcPromise)
    if (kbfsMount && fs.existsSync(kbfsMount)) {
      status.installStatus = RPCTypes.installInstallStatus.installed
      status.installAction = RPCTypes.installInstallAction.none
      status.kextStarted = true
    }
  }
  yield Saga.put(FsGen.createFuseStatusResult({prevStatus, status}))
}

function* installFuseSaga(): Saga.SagaGenerator<any, any> {
  const result: RPCTypes.InstallResult = yield Saga.call(RPCTypes.installInstallFuseRpcPromise)
  const fuseResults =
    result && result.componentResults ? result.componentResults.filter(c => c.name === 'fuse') : []
  const kextPermissionError =
    fuseResults.length > 0 && fuseResults[0].exitCode === Constants.ExitCodeFuseKextPermissionError

  if (kextPermissionError) {
    // Add a small delay here, since on 10.13 the OS will be a little laggy
    // when showing a kext permission error.
    yield Saga.delay(1e3)
  }

  yield Saga.put(FsGen.createInstallFuseResult({kextPermissionError}))
  yield Saga.put(FsGen.createFuseStatus())
  yield Saga.put(FsGen.createSetFlags({fuseInstalling: false}))
  // TODO: do something like uninstallConfirmSaga here
}

const uninstallKBFSConfirm = (action: FsGen.UninstallKBFSConfirmPayload) =>
  new Promise((resolve, reject) =>
    SafeElectron.getDialog().showMessageBox(
      null,
      {
        buttons: ['Remove & Restart', 'Cancel'],
        detail: `Are you sure you want to remove Keybase from ${fileUIName} and restart the app?`,
        message: `Remove Keybase from ${fileUIName}`,
        type: 'question',
      },
      resp => resolve(resp)
    )
  )

const uninstallKBFSConfirmSuccess = resp =>
  resp
    ? undefined
    : Saga.sequentially([
        Saga.call(RPCTypes.installUninstallKBFSRpcPromise),
        Saga.call(() => {
          // Restart since we had to uninstall KBFS and it's needed by the service (for chat)
          SafeElectron.getApp().relaunch()
          SafeElectron.getApp().exit(0)
        }),
      ])

const openSecurityPreferences = () =>
  Saga.call(
    () =>
      new Promise((resolve, reject) => {
        SafeElectron.getShell().openExternal(
          'x-apple.systempreferences:com.apple.preference.security?General',
          {activate: true},
          err => {
            if (err) {
              reject(err)
              return
            }
            logger.info('Opened Security Preferences')
            resolve()
          }
        )
      })
  )

// Invoking the cached installer package has to happen from the topmost process
// or it won't be visible to the user. The service also does this to support command line
// operations.
function installCachedDokan() {
  return new Promise((resolve, reject) => {
    logger.info('Invoking dokan installer')
    const dokanPath = path.resolve(String(process.env.LOCALAPPDATA), 'Keybase', 'DokanSetup_redist.exe')
    try {
      execFileSync(dokanPath, [])
    } catch (err) {
      logger.error('installCachedDokan caught', err)
      reject(err)
      return
    }
    // restart the service, particularly kbfsdokan
    // based on desktop/app/start-win-service.js
    const binPath = path.resolve(String(process.env.LOCALAPPDATA), 'Keybase', 'keybase.exe')
    if (!binPath) {
      return
    }
    const rqPath = binPath.replace('keybase.exe', 'keybaserq.exe')
    const args = [binPath, 'ctl', 'restart']

    spawn(rqPath, args, {
      detached: true,
      stdio: 'ignore',
    })

    resolve()
  })
}

function uninstallDokan(uninstallString: string): Promise<boolean> {
  return new Promise((resolve, reject) => {
    logger.info('Invoking dokan uninstaller')
    try {
      execSync(uninstallString, {windowsHide: true})
      resolve(true)
    } catch (err) {
      logger.error('uninstallDokan caught', err)
      reject(err)
    }
  })
}

function installDokanSaga() {
  return Saga.call(installCachedDokan)
}

function* uninstallDokanSaga() {
  const state: TypedState = yield Saga.select()
  const uninstallString = Constants.kbfsUninstallString(state)
  if (uninstallString) {
    yield Saga.call(uninstallDokan, uninstallString)
    yield Saga.put(FsGen.createFuseStatus())
  }
}

const openAndUploadToPromise = (state: TypedState, action: FsGen.OpenAndUploadPayload) =>
  new Promise((resolve, reject) =>
    SafeElectron.getDialog().showOpenDialog(
      SafeElectron.getCurrentWindowFromRemote(),
      {
        title: 'Select a file or folder to upload',
        properties: [
          'multiSelections',
          ...(['file', 'both'].includes(action.payload.type) ? ['openFile'] : []),
          ...(['directory', 'both'].includes(action.payload.type) ? ['openDirectory'] : []),
        ],
      },
      filePaths => resolve(filePaths || [])
    )
  )

const openAndUpload = (state: TypedState, action: FsGen.OpenAndUploadPayload) =>
  Saga.call(function*() {
    const localPaths = yield Saga.call(openAndUploadToPromise, state, action)
    yield Saga.all(
      localPaths.map(localPath =>
        Saga.put(FsGen.createUpload({localPath, parentPath: action.payload.parentPath}))
      )
    )
  })

function* platformSpecificSaga(): Saga.SagaGenerator<any, any> {
  yield Saga.safeTakeEveryPure(FsGen.openInFileUI, openInFileUISaga)
  yield Saga.safeTakeEvery(FsGen.fuseStatus, fuseStatusSaga)
  yield Saga.safeTakeEveryPure(FsGen.fuseStatusResult, fuseStatusResultSaga)
  yield Saga.safeTakeEveryPure(FsGen.installKBFS, RPCTypes.installInstallKBFSRpcPromise, installKBFSSuccess)
  yield Saga.actionToAction(FsGen.openAndUpload, openAndUpload)
  if (isWindows) {
    yield Saga.safeTakeEveryPure(FsGen.installFuse, installDokanSaga)
    yield Saga.safeTakeEvery(FsGen.uninstallKBFSConfirm, uninstallDokanSaga)
  } else {
    yield Saga.safeTakeEvery(FsGen.installFuse, installFuseSaga)
    yield Saga.safeTakeEveryPure(FsGen.uninstallKBFSConfirm, uninstallKBFSConfirm, uninstallKBFSConfirmSuccess)
  }
  yield Saga.safeTakeEveryPure(FsGen.openSecurityPreferences, openSecurityPreferences)
}

export default platformSpecificSaga
