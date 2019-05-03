package webrtc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"sync"
	"testing"
)

var testData = []byte("this is some test data")

func TestPingPongDetach(t *testing.T) {
	label := "test-channel"

	// Use Detach data channels mode
	s := SettingEngine{}
	s.DetachDataChannels()
	api := NewAPI(WithSettingEngine(s))

	// Set up two peer connections.
	config := Configuration{}
	pca, err := api.NewPeerConnection(config)
	if err != nil {
		t.Fatal(err)
	}
	pcb, err := api.NewPeerConnection(config)
	if err != nil {
		t.Fatal(err)
	}

	defer pca.Close()
	defer pcb.Close()

	var wg sync.WaitGroup

	dcChan := make(chan *DataChannel)
	pcb.OnDataChannel(func(dc *DataChannel) {
		if dc.Label() == label {
			dcChan <- dc
		} else {
			fmt.Printf("Uknown datachannel opened: %s\n", dc.Label())
		}
	})

	wg.Add(1)
	go func() {
		defer wg.Done()

		fmt.Println("Waiting for OnDataChannel")
		attached := <-dcChan
		fmt.Println("OnDataChannel was called")
		open := make(chan struct{})
		attached.OnOpen(func() {
			open <- struct{}{}
		})
		<-open
		dc, err := attached.Detach()
		fmt.Println("post: pt1")
		if err != nil {
			fmt.Printf("Detach failed: %s", err.Error())
			t.Fatal(err)
		}
		fmt.Println("post: pt2")
		defer dc.Close()

		fmt.Println("Waiting for ping...")
		msg, err := ioutil.ReadAll(dc)
		fmt.Printf("Received ping! \"%s\"\n", string(msg))
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println("Sending pong")
		if _, err := dc.Write(msg); err != nil {
			t.Fatal(err)
		}
		fmt.Println("Sent pong")
	}()

	if err := signalPair(pca, pcb); err != nil {
		t.Fatal(err)
	}

	attached, err := pca.CreateDataChannel(label, nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Waiting for data channel to open")
	open := make(chan struct{})
	attached.OnOpen(func() {
		open <- struct{}{}
	})
	<-open
	fmt.Println("data channel opened")
	dc, err := attached.Detach()
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		data := []byte(fmt.Sprintf("%s - %d", testData, 0))
		fmt.Println("Sending ping...")
		if _, err := dc.Write(data); err != nil {
			t.Fatal(err)
		}
		fmt.Println("Sent ping")

		dc.Close()

		fmt.Println("Wating for pong")
		ret, err := ioutil.ReadAll(dc)
		if err != nil {
			fmt.Println("Error here")
			t.Fatal(err)
			return
		}
		fmt.Printf("Received pong! \"%s\"\n", string(ret))
		if !bytes.Equal(data, ret) {
			fmt.Println("Received pong BAD!")
			t.Errorf("expected %q, got %q", string(data), string(ret))
		} else {
			fmt.Println("Received pong GOOD!")
		}
	}()

	wg.Wait()
}

/*
func TestPingPongNoDetach(t *testing.T) {
	// streams := 1

	// Use Detach data channels mode
	s := SettingEngine{}
	//s.DetachDataChannels()
	api := NewAPI(WithSettingEngine(s))

	// Set up two peer connections.
	config := Configuration{}
	pca, err := api.NewPeerConnection(config)
	if err != nil {
		t.Fatal(err)
	}
	pcb, err := api.NewPeerConnection(config)
	if err != nil {
		t.Fatal(err)
	}

	var dca, dcb *DataChannel
	doneCh := make(chan struct{})

	pcb.OnDataChannel(func(dc *DataChannel) {
		fmt.Printf("pcb: new datachannel: %s\n", dc.Label())

		dcb = dc
		// Register channel opening handling
		dcb.OnOpen(func() {
			fmt.Println("pcb: datachannel opened")
		})

		// Register the OnMessage to handle incoming messages
		fmt.Println("pcb: registering onMessage callback")
		dcb.OnMessage(func(dcMsg DataChannelMessage) {
			fmt.Printf("pcb: received ping: %s\n", string(dcMsg.Data))

			if err := dcb.SendText(string(dcMsg.Data)); err != nil {
				t.Fatal(err)
			}
			fmt.Println("pcb: sent pong")
		})
	})

	dca, err = pca.CreateDataChannel("test-channel", nil)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("pca: waiting for data channel to open")
	dca.OnOpen(func() {
		fmt.Println("pca: data channel opened")
		data := []byte(fmt.Sprintf("%s - %d", testData, 0))
		if err := dca.SendText(string(data)); err != nil {
			t.Fatal(err)
		}
		fmt.Println("pca: sent ping")
	})

	// Register the OnMessage to handle incoming messages
	fmt.Println("pca: registering onMessage callback")
	dca.OnMessage(func(dcMsg DataChannelMessage) {
		fmt.Printf("pca: received pong: %s\n", string(dcMsg.Data))
		close(doneCh)
	})

	if err := signalPair(pca, pcb); err != nil {
		t.Fatal(err)
	}

	<-doneCh

	dca.Close()
	dcb.Close()

	pca.Close()
	pcb.Close()
}
*/
