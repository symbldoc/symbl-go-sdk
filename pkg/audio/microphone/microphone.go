// Copyright 2022 Symbl.ai SDK contributors. All Rights Reserved.
// SPDX-License-Identifier: MIT

package microphone

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/gordonklaus/portaudio"
	klog "k8s.io/klog/v2"
)

func Initialize(cfg AudioConfig) (*Microphone, error) {
	m := &Microphone{
		sig:    make(chan os.Signal, 1),
		intBuf: make([]int16, 1024),
	}
	signal.Notify(m.sig, os.Interrupt, os.Kill)

	portaudio.Initialize()

	stream, err := portaudio.OpenDefaultStream(cfg.InputChannels, 0, float64(cfg.SamplingRate), len(m.intBuf), m.intBuf)
	if err != nil {
		klog.V(1).Infof("OpenDefaultStream failed. Err: %v\n", err)
		return nil, err
	}

	m.stream = stream
	klog.V(3).Infof("OpenDefaultStream succeded\n")
	return m, nil
}

func (m *Microphone) Start() error {
	err := m.stream.Start()
	if err != nil {
		klog.V(1).Infof("Mic failed to start. Err: %v\n", err)
		return err
	}

	klog.V(3).Infof("Start() succeded\n")
	return nil
}

func (m *Microphone) Read() ([]int16, error) {
	err := m.stream.Read()
	if err != nil {
		klog.V(1).Infof("stream.Read failed. Err: %v\n", err)
		return nil, err
	}

	buf := make([]int16, 1024)
	byteCopied := copy(buf, m.intBuf)
	klog.V(5).Infof("stream.Read bytes copied: %d\n", byteCopied)
	return buf, nil
}

func (m *Microphone) Stream(w io.Writer) error {
	for {
		err := m.stream.Read()
		if err != nil {
			klog.V(1).Infof("stream.Read failed. Err: %v\n", err)
			return err
		}

		byteCount, err := w.Write(int16ToLittleEndianByte(m.intBuf))
		if err != nil {
			klog.V(1).Infof("w.Write failed. Err: %v\n", err)
			return err
		}
		klog.V(5).Infof("io.Writer succeeded. Bytes written: %d\n", byteCount)

		select {
		case <-m.sig:
			return nil
		default:
		}
	}

	return nil
}

func (m *Microphone) Stop() error {
	err := m.stream.Stop()
	if err != nil {
		klog.V(1).Infof("stream.Stop failed. Err: %v\n", err)
		return err
	}
	return nil
}

func Teardown() {
	portaudio.Terminate()
}

func int16ToLittleEndianByte(f []int16) []byte {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, f)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return buf.Bytes()
}
