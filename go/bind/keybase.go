// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package keybase

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/trace"
	"sync"
	"time"

	"github.com/keybase/client/go/chat"
	"github.com/keybase/client/go/chat/globals"

	"strings"

	"github.com/keybase/client/go/externals"
	"github.com/keybase/client/go/libkb"
	"github.com/keybase/client/go/logger"
	"github.com/keybase/client/go/protocol/chat1"
	"github.com/keybase/client/go/protocol/gregor1"
	"github.com/keybase/client/go/protocol/keybase1"
	"github.com/keybase/client/go/service"
	"github.com/keybase/client/go/uidmap"
	"github.com/keybase/go-framed-msgpack-rpc/rpc"
	"github.com/keybase/kbfs/env"
	"github.com/keybase/kbfs/fsrpc"
	"github.com/keybase/kbfs/libgit"
	"github.com/keybase/kbfs/libkbfs"
	"github.com/keybase/kbfs/simplefs"
)

var kbCtx *libkb.GlobalContext
var kbChatCtx *globals.ChatContext
var conn net.Conn
var startOnce sync.Once
var logSendContext libkb.LogSendContext
var kbfsConfig libkbfs.Config

type PushNotifier interface {
	LocalNotification(ident string, msg string, badgeCount int, soundName string, convID string, typ string)
}

type ExternalDNSNSFetcher interface {
	GetServers() []byte
}

type dnsNSFetcher struct {
	externalFetcher ExternalDNSNSFetcher
}

func newDNSNSFetcher(d ExternalDNSNSFetcher) dnsNSFetcher {
	return dnsNSFetcher{
		externalFetcher: d,
	}
}

func (d dnsNSFetcher) processExternalResult(raw []byte) []string {
	return strings.Split(string(raw), ",")
}

func (d dnsNSFetcher) GetServers() []string {
	if d.externalFetcher != nil {
		return d.processExternalResult(d.externalFetcher.GetServers())
	}
	return getDNSServers()
}

var _ libkb.DNSNameServerFetcher = dnsNSFetcher{}

// InitOnce runs the Keybase services (only runs one time)
func InitOnce(homeDir string, logFile string, runModeStr string, accessGroupOverride bool,
	dnsNSFetcher ExternalDNSNSFetcher) {
	startOnce.Do(func() {
		if err := Init(homeDir, logFile, runModeStr, accessGroupOverride, dnsNSFetcher); err != nil {
			kbCtx.Log.Errorf("Init error: %s", err)
		}
	})
}

// Init runs the Keybase services
func Init(homeDir string, logFile string, runModeStr string, accessGroupOverride bool,
	externalDNSNSFetcher ExternalDNSNSFetcher) error {
	fmt.Println("Go: Initializing")
	if logFile != "" {
		fmt.Printf("Go: Using log: %s\n", logFile)
	}

	// Reduce OS threads on mobile so we don't have too much contention with JS thread
	oldProcs := runtime.GOMAXPROCS(0)
	newProcs := oldProcs / 2
	runtime.GOMAXPROCS(newProcs)
	fmt.Printf("Go: setting GOMAXPROCS to: %d previous: %d\n", newProcs, oldProcs)

	startTrace(logFile)

	dnsNSFetcher := newDNSNSFetcher(externalDNSNSFetcher)
	dnsServers := dnsNSFetcher.GetServers()
	for _, srv := range dnsServers {
		fmt.Printf("Go: DNS Server: %s\n", srv)
	}

	kbCtx = libkb.NewGlobalContext()
	kbCtx.Init()
	kbCtx.SetServices(externals.GetServices())

	// 10k uid -> FullName cache entries allowed
	kbCtx.SetUIDMapper(uidmap.NewUIDMap(10000))
	usage := libkb.Usage{
		Config:    true,
		API:       true,
		KbKeyring: true,
	}
	runMode, err := libkb.StringToRunMode(runModeStr)
	if err != nil {
		return err
	}
	config := libkb.AppConfig{
		HomeDir:                        homeDir,
		LogFile:                        logFile,
		RunMode:                        runMode,
		Debug:                          true,
		LocalRPCDebug:                  "",
		VDebugSetting:                  "mobile", // use empty string for same logging as desktop default
		SecurityAccessGroupOverride:    accessGroupOverride,
		ChatInboxSourceLocalizeThreads: 5,
	}
	err = kbCtx.Configure(config, usage)
	if err != nil {
		return err
	}

	svc := service.NewService(kbCtx, false)
	err = svc.StartLoopbackServer()
	if err != nil {
		return err
	}
	kbCtx.SetService()
	uir := service.NewUIRouter(kbCtx)
	kbCtx.SetUIRouter(uir)
	kbCtx.SetDNSNameServerFetcher(dnsNSFetcher)
	svc.SetupCriticalSubServices()
	svc.RunBackgroundOperations(uir)
	kbChatCtx = svc.ChatContextified.ChatG()

	serviceLog := config.GetLogFile()
	logs := libkb.Logs{
		Service: serviceLog,
	}

	logSendContext = libkb.LogSendContext{
		Contextified: libkb.NewContextified(kbCtx),
		Logs:         logs,
	}

	// open the connection
	err = Reset()
	if err != nil {
		return err
	}

	go func() {
		kbfsCtx := env.NewContextFromGlobalContext(kbCtx)
		kbfsParams := libkbfs.DefaultInitParams(kbfsCtx)
		// Setting this flag will enable KBFS debug logging to always
		// be true in a mobile setting. Change these back to the
		// commented-out values if we need to make a mobile release
		// before KBFS-on-mobile is ready.
		kbfsParams.Debug = true                         // false
		kbfsParams.Mode = libkbfs.InitConstrainedString // libkbfs.InitMinimalString
		kbfsConfig, _ = libkbfs.Init(
			context.Background(), kbfsCtx, kbfsParams, serviceCn{}, func() {},
			kbCtx.Log)
	}()

	return nil
}

type serviceCn struct {
	ctx *libkb.GlobalContext
}

func (s serviceCn) NewKeybaseService(config libkbfs.Config, params libkbfs.InitParams, ctx libkbfs.Context, log logger.Logger) (libkbfs.KeybaseService, error) {
	// TODO: plumb the func somewhere it can be called on shutdown?
	gitrpc, _ := libgit.NewRPCHandlerWithCtx(
		ctx, config, nil)
	additionalProtocols := []rpc.Protocol{
		keybase1.SimpleFSProtocol(
			simplefs.NewSimpleFS(ctx.GetGlobalContext(), config)),
		keybase1.KBFSGitProtocol(gitrpc),
		keybase1.FsProtocol(fsrpc.NewFS(config, log)),
	}
	keybaseService := libkbfs.NewKeybaseDaemonRPC(
		config, ctx, log, true, additionalProtocols)
	return keybaseService, nil
}

func (s serviceCn) NewCrypto(config libkbfs.Config, params libkbfs.InitParams, ctx libkbfs.Context, log logger.Logger) (libkbfs.Crypto, error) {
	return libkbfs.NewCryptoClientRPC(config, ctx), nil
}

func (s serviceCn) NewChat(config libkbfs.Config, params libkbfs.InitParams, ctx libkbfs.Context, log logger.Logger) (libkbfs.Chat, error) {
	return libkbfs.NewChatRPC(config, ctx), nil
}

// LogSend sends a log to Keybase
func LogSend(status string, feedback string, sendLogs bool, uiLogPath, traceDir string) (string, error) {
	logSendContext.Logs.Desktop = uiLogPath
	logSendContext.Logs.Trace = traceDir
	env := kbCtx.Env
	return logSendContext.LogSend(status, feedback, sendLogs, 10*1024*1024, env.GetUID(), env.GetInstallID())
}

// WriteB64 sends a base64 encoded msgpack rpc payload
func WriteB64(str string) error {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return fmt.Errorf("Base64 decode error: %s; %s", err, str)
	}
	n, err := conn.Write(data)
	if err != nil {
		return fmt.Errorf("Write error: %s", err)
	}
	if n != len(data) {
		return errors.New("Did not write all the data")
	}
	return nil
}

const targetBufferSize = 50 * 1024

// bufferSize must be divisible by 3 to ensure that we don't split
// our b64 encode across a payload boundary if we go over our buffer
// size.
const bufferSize = targetBufferSize - (targetBufferSize % 3)

// buffer for the conn.Read
var buffer = make([]byte, bufferSize)

// ReadB64 is a blocking read for base64 encoded msgpack rpc data.
// It is called serially by the mobile run loops.
func ReadB64() (string, error) {
	n, err := conn.Read(buffer)
	if n > 0 && err == nil {
		str := base64.StdEncoding.EncodeToString(buffer[0:n])
		return str, nil
	}

	if err != nil {
		// Attempt to fix the connection
		Reset()
		return "", fmt.Errorf("Read error: %s", err)
	}

	return "", nil
}

// Reset resets the socket connection
func Reset() error {
	if conn != nil {
		conn.Close()
	}

	var err error
	conn, err = kbCtx.LoopbackListener.Dial()
	if err != nil {
		return fmt.Errorf("Socket error: %s", err)
	}
	return nil
}

// ForceGC Forces a gc
func ForceGC() {
	fmt.Printf("Flushing global caches\n")
	kbCtx.FlushCaches()
	fmt.Printf("Done flushing global caches\n")

	fmt.Printf("Starting force gc\n")
	debug.FreeOSMemory()
	fmt.Printf("Done force gc\n")
}

// Version returns semantic version string
func Version() string {
	return libkb.VersionString()
}

func SetAppStateForeground() {
	defer kbCtx.Trace("SetAppStateForeground", func() error { return nil })()
	kbCtx.AppState.Update(keybase1.AppState_FOREGROUND)
}
func SetAppStateBackground() {
	defer kbCtx.Trace("SetAppStateBackground", func() error { return nil })()
	kbCtx.AppState.Update(keybase1.AppState_BACKGROUND)
}
func SetAppStateInactive() {
	defer kbCtx.Trace("SetAppStateInactive", func() error { return nil })()
	kbCtx.AppState.Update(keybase1.AppState_INACTIVE)
}
func SetAppStateBackgroundActive() {
	defer kbCtx.Trace("SetAppStateBackgroundActive", func() error { return nil })()
	kbCtx.AppState.Update(keybase1.AppState_BACKGROUNDACTIVE)
}

func BackgroundSync() {
	defer kbCtx.Trace("BackgroundSync", func() error { return nil })()

	// Skip the sync if we aren't in the background
	if state := kbCtx.AppState.State(); state != keybase1.AppState_BACKGROUND {
		kbCtx.Log.Debug("BackgroundSync: skipping, app not in background state: %v", state)
		return
	}

	nextState := keybase1.AppState_BACKGROUNDACTIVE
	kbCtx.AppState.Update(nextState)
	doneCh := make(chan struct{})
	go func() {
		defer func() { close(doneCh) }()
		select {
		case state := <-kbCtx.AppState.NextUpdate(&nextState):
			// if literally anything happens, let's get out of here
			kbCtx.Log.Debug("BackgroundSync: bailing out early, appstate change: %v", state)
			return
		case <-time.After(10 * time.Second):
			kbCtx.AppState.Update(keybase1.AppState_BACKGROUND)
			return
		}
	}()
	<-doneCh
}

func HandleBackgroundNotification(strConvID string, intMembersType int, intMessageID int,
	pushID string, badgeCount int, unixTime int, body string, pusher PushNotifier) (err error) {
	gc := globals.NewContext(kbCtx, kbChatCtx)
	ctx := chat.Context(context.Background(), gc,
		keybase1.TLFIdentifyBehavior_CHAT_GUI, nil, chat.NewCachingIdentifyNotifier(gc))
	defer kbCtx.CTrace(ctx, fmt.Sprintf("HandleBackgroundNotification(%s,%d,%d,%s,%d,%d)",
		strConvID, intMembersType, intMessageID, pushID, badgeCount, unixTime),
		func() error { return err })()
	uid := gregor1.UID(kbCtx.Env.GetUID().ToBytes())
	if !kbCtx.ActiveDevice.HaveKeys() {
		return libkb.LoginRequiredError{}
	}
	age := time.Since(time.Unix(int64(unixTime), 0))
	if age >= 15*time.Second {
		kbCtx.Log.CDebugf(ctx, "HandleBackgroundNotification: stale notification: %v", age)
		return errors.New("stale notification")
	}

	bConvID, err := hex.DecodeString(strConvID)
	if err != nil {
		kbCtx.Log.CDebugf(ctx, "HandleBackgroundNotification: invalid convID: %s msg: %s", strConvID,
			err)
		return err
	}
	convID := chat1.ConversationID(bConvID)
	membersType := chat1.ConversationMembersType(intMembersType)
	msg, err := kbCtx.ChatHelper.UnboxMobilePushNotification(ctx, uid, convID, membersType, []string{pushID},
		body)
	if err != nil {
		kbCtx.Log.CDebugf(ctx, "HandleBackgroundNotification: failed to unbox: %s", err)
		return err
	}

	// Send up the local notification with our message
	id := fmt.Sprintf("%s:%d", strConvID, intMessageID)
	pusher.LocalNotification(id, msg, badgeCount, "keybasemessage.wav", strConvID, "chat.newmessage")
	// Hit the remote server to let it know we succeeded in showing something useful
	kbCtx.ChatHelper.AckMobileNotificationSuccess(ctx, []string{pushID})
	return nil
}

// AppWillExit is called reliably on iOS when the app is about to terminate
// not as reliably on android
func AppWillExit() {
	defer kbCtx.Trace("AppWillExit", func() error { return nil })()
	kbCtx.AppState.Update(keybase1.AppState_BACKGROUNDFINAL)
}

// AppDidEnterBackground notifies the service that the app is in the background
// [iOS] returning true will request about ~3mins from iOS to continue execution
func AppDidEnterBackground() bool {
	defer kbCtx.Trace("AppDidEnterBackground", func() error { return nil })()
	SetAppStateBackground()
	return false
}

func startTrace(logFile string) {
	if os.Getenv("KEYBASE_TRACE_MOBILE") != "1" {
		return
	}

	tname := filepath.Join(filepath.Dir(logFile), "svctrace.out")
	f, err := os.Create(tname)
	if err != nil {
		fmt.Printf("error creating %s\n", tname)
		return
	}
	fmt.Printf("Go: starting trace %s\n", tname)
	trace.Start(f)
	go func() {
		fmt.Printf("Go: sleeping 30s for trace\n")
		time.Sleep(30 * time.Second)
		fmt.Printf("Go: stopping trace %s\n", tname)
		trace.Stop()
		time.Sleep(5 * time.Second)
		fmt.Printf("Go: trace stopped\n")
	}()
}
