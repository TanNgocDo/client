// @flow
// NOTE: This file is GENERATED from json files in actions/json. Run 'yarn build-actions' to regenerate
/* eslint-disable no-unused-vars,prettier/prettier,no-use-before-define */

import * as I from 'immutable'
import * as RPCTypes from '../constants/types/rpc-gen'
import * as Types from '../constants/types/devices'
import * as Constants from '../constants/devices'
import HiddenString from '../util/hidden-string'

// Constants
export const resetStore = 'common:resetStore' // not a part of devices but is handled by every reducer. NEVER dispatch this
export const typePrefix = 'devices:'
export const endangeredTLFsLoaded = typePrefix + 'endangeredTLFsLoaded'
export const load = typePrefix + 'load'
export const loaded = typePrefix + 'loaded'
export const paperKeyCreated = typePrefix + 'paperKeyCreated'
export const revoke = typePrefix + 'revoke'
export const revoked = typePrefix + 'revoked'
export const showDevicePage = typePrefix + 'showDevicePage'
export const showPaperKeyPage = typePrefix + 'showPaperKeyPage'
export const showRevokePage = typePrefix + 'showRevokePage'

// Payload Types
type _EndangeredTLFsLoadedPayload = $ReadOnly<{|
  deviceID: Types.DeviceID,
  tlfs: Array<string>,
|}>
type _LoadPayload = void
type _LoadedPayload = $ReadOnly<{|devices: Array<Types.Device>|}>
type _PaperKeyCreatedPayload = $ReadOnly<{|paperKey: HiddenString|}>
type _RevokePayload = $ReadOnly<{|deviceID: Types.DeviceID|}>
type _RevokedPayload = $ReadOnly<{|
  deviceID: Types.DeviceID,
  wasCurrentDevice: boolean,
  deviceName: string,
|}>
type _ShowDevicePagePayload = $ReadOnly<{|deviceID: Types.DeviceID|}>
type _ShowPaperKeyPagePayload = void
type _ShowRevokePagePayload = $ReadOnly<{|deviceID: Types.DeviceID|}>

// Action Creators
export const createEndangeredTLFsLoaded = (payload: _EndangeredTLFsLoadedPayload) => ({error: false, payload, type: endangeredTLFsLoaded})
export const createLoad = (payload: _LoadPayload) => ({error: false, payload, type: load})
export const createLoaded = (payload: _LoadedPayload) => ({error: false, payload, type: loaded})
export const createPaperKeyCreated = (payload: _PaperKeyCreatedPayload) => ({error: false, payload, type: paperKeyCreated})
export const createRevoke = (payload: _RevokePayload) => ({error: false, payload, type: revoke})
export const createRevoked = (payload: _RevokedPayload) => ({error: false, payload, type: revoked})
export const createShowDevicePage = (payload: _ShowDevicePagePayload) => ({error: false, payload, type: showDevicePage})
export const createShowPaperKeyPage = (payload: _ShowPaperKeyPagePayload) => ({error: false, payload, type: showPaperKeyPage})
export const createShowRevokePage = (payload: _ShowRevokePagePayload) => ({error: false, payload, type: showRevokePage})

// Action Payloads
export type EndangeredTLFsLoadedPayload = $Call<typeof createEndangeredTLFsLoaded, _EndangeredTLFsLoadedPayload>
export type LoadPayload = $Call<typeof createLoad, _LoadPayload>
export type LoadedPayload = $Call<typeof createLoaded, _LoadedPayload>
export type PaperKeyCreatedPayload = $Call<typeof createPaperKeyCreated, _PaperKeyCreatedPayload>
export type RevokePayload = $Call<typeof createRevoke, _RevokePayload>
export type RevokedPayload = $Call<typeof createRevoked, _RevokedPayload>
export type ShowDevicePagePayload = $Call<typeof createShowDevicePage, _ShowDevicePagePayload>
export type ShowPaperKeyPagePayload = $Call<typeof createShowPaperKeyPage, _ShowPaperKeyPagePayload>
export type ShowRevokePagePayload = $Call<typeof createShowRevokePage, _ShowRevokePagePayload>

// All Actions
// prettier-ignore
export type Actions =
  | EndangeredTLFsLoadedPayload
  | LoadPayload
  | LoadedPayload
  | PaperKeyCreatedPayload
  | RevokePayload
  | RevokedPayload
  | ShowDevicePagePayload
  | ShowPaperKeyPagePayload
  | ShowRevokePagePayload
  | {type: 'common:resetStore', payload: void}
