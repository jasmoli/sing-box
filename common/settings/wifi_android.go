//go:build android && cgo

package settings

/*
#include <stdlib.h>
extern int box_wifi_state(char *ssid, int ssidLength, unsigned char *bssid);
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common/logger"
)

const androidWIFIPollInterval = 3 * time.Second

type androidWIFIMonitor struct {
	logger   logger.ContextLogger
	callback func(adapter.WIFIState)
	cancel   context.CancelFunc
}

func newAndroidWIFIMonitor(logger logger.ContextLogger, callback func(adapter.WIFIState)) (WIFIMonitor, error) {
	return &androidWIFIMonitor{logger: logger, callback: callback}, nil
}

func (m *androidWIFIMonitor) readState() (adapter.WIFIState, error) {
	var ssidBuffer [33]C.char
	var bssidBuffer [6]C.uchar
	result := C.box_wifi_state(&ssidBuffer[0], C.int(len(ssidBuffer)), &bssidBuffer[0])
	if result != 0 {
		return adapter.WIFIState{}, syscall.Errno(result)
	}
	ssid := C.GoString(&ssidBuffer[0])
	if ssid == "" {
		return adapter.WIFIState{}, nil
	}
	return adapter.WIFIState{
		SSID: ssid,
		BSSID: fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
			bssidBuffer[0], bssidBuffer[1], bssidBuffer[2],
			bssidBuffer[3], bssidBuffer[4], bssidBuffer[5]),
	}, nil
}

func (m *androidWIFIMonitor) ReadWIFIState(ctx context.Context) adapter.WIFIState {
	state, _ := m.readState()
	return state
}

func (m *androidWIFIMonitor) Start() error {
	if m.callback == nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	state, err := m.readState()
	if err != nil {
		m.logger.Warn("read initial WIFI state: ", err)
	} else if state.SSID == "" {
		m.logger.Info("Android WIFI monitor started, not connected")
	} else {
		m.logger.Info("Android WIFI monitor started, SSID=", state.SSID, ", BSSID=", state.BSSID)
	}
	m.callback(state)
	go m.poll(ctx, state)
	return nil
}

func (m *androidWIFIMonitor) poll(ctx context.Context, lastState adapter.WIFIState) {
	ticker := time.NewTicker(androidWIFIPollInterval)
	defer ticker.Stop()
	var lastError error
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state, err := m.readState()
			if err != nil {
				if !errors.Is(err, lastError) {
					m.logger.Warn("read WIFI state: ", err)
					lastError = err
				}
				continue
			}
			if lastError != nil {
				m.logger.Info("read WIFI state recovered")
				lastError = nil
			}
			if state != lastState {
				lastState = state
				m.callback(state)
			}
		}
	}
}

func (m *androidWIFIMonitor) Close() error {
	if m.cancel != nil {
		m.cancel()
	}
	return nil
}
