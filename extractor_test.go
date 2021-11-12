package extractor

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/types"
)

func TestExtractorInitOutput(t *testing.T) {
	ex := NewExtractorService(nil, &Config{})
	err := ex.initStreamOutput()
	assert.NoError(t, err)
	assert.IsType(t, &consoleWriter{}, ex.writer)

	ex = NewExtractorService(nil, &Config{OutputFile: "STDOUT"})
	err = ex.initStreamOutput()
	assert.NoError(t, err)
	assert.IsType(t, &consoleWriter{}, ex.writer)

	ex = NewExtractorService(nil, &Config{OutputFile: "STDERR"})
	err = ex.initStreamOutput()
	assert.NoError(t, err)
	assert.IsType(t, &consoleWriter{}, ex.writer)

	ex = NewExtractorService(nil, &Config{OutputFile: fmt.Sprintf("/tmp/%v", time.Now().Unix())})
	err = ex.initStreamOutput()
	assert.NoError(t, err)
	assert.IsType(t, &fileWriter{}, ex.writer)

	ex = NewExtractorService(nil, &Config{
		OutputFile: fmt.Sprintf("/tmp/%v", time.Now().Unix()),
		Bundle:     true,
	})
	err = ex.initStreamOutput()
	assert.NoError(t, err)
	assert.IsType(t, &bundleWriter{}, ex.writer)
}

func TestIndexBlock(t *testing.T) {
	examples := []struct {
		input    types.EventDataNewBlock
		expected string
		err      string
	}{
		{
			input: types.EventDataNewBlock{
				Block: &types.Block{
					Header: types.Header{
						ChainID: "chain-id",
						Height:  1,
						Time:    time.Unix(1634674166, 0),
					},
					LastCommit: &types.Commit{
						BlockID: types.BlockID{
							Hash: []byte{},
							PartSetHeader: types.PartSetHeader{
								Total: 1,
								Hash:  []byte{},
							},
						},
					},
				},
			},
			expected: "DMLOG BLOCK 1 1634674166000 CiYKHAoAEghjaGFpbi1pZBgBIgYI9su8iwYqAhIAcgAiBhoEEgIIARIA\n",
		},
		{
			input: types.EventDataNewBlock{
				Block: &types.Block{
					Header: types.Header{
						ChainID: "chain-id",
						Height:  2,
						Time:    time.Unix(1634674166, 0),
					},
					LastCommit: &types.Commit{
						BlockID: types.BlockID{
							Hash: []byte{},
							PartSetHeader: types.PartSetHeader{
								Total: 1,
								Hash:  []byte{},
							},
						},
					},
				},
				ResultBeginBlock: abci.ResponseBeginBlock{
					Events: []abci.Event{
						{
							Type: "eventType1",
							Attributes: []abci.EventAttribute{
								{Key: []byte("key1"), Value: []byte("value1")},
							},
						},
						{
							Type: "eventType2",
							Attributes: []abci.EventAttribute{
								{Key: []byte("key1"), Value: []byte("value1")},
							},
						},
					},
				},
				ResultEndBlock: abci.ResponseEndBlock{
					Events: []abci.Event{
						{
							Type: "eventType1",
							Attributes: []abci.EventAttribute{
								{Key: []byte("key1"), Value: []byte("value1")},
							},
						},
						{
							Type: "eventType2",
							Attributes: []abci.EventAttribute{
								{Key: []byte("key1"), Value: []byte("value1")},
							},
						},
					},
				},
			},
			expected: "DMLOG BLOCK 2 1634674166000 CiYKHAoAEghjaGFpbi1pZBgCIgYI9su8iwYqAhIAcgAiBhoEEgIIARIAGjwKHAoKZXZlbnRUeXBlMRIOCgRrZXkxEgZ2YWx1ZTEKHAoKZXZlbnRUeXBlMhIOCgRrZXkxEgZ2YWx1ZTEiPhIAGhwKCmV2ZW50VHlwZTESDgoEa2V5MRIGdmFsdWUxGhwKCmV2ZW50VHlwZTISDgoEa2V5MRIGdmFsdWUx\n",
		},
	}

	for _, ex := range examples {
		output := bytes.NewBuffer(nil)
		extractor := NewExtractorService(nil, nil)
		extractor.writer = NewConsoleWriter(output)

		err := extractor.indexBlock(ex.input)
		if err != nil {
			assert.Equal(t, err.Error(), ex.err)
		}
		assert.Equal(t, ex.expected, output.String())
	}
}

func TestIndexTx(t *testing.T) {
	examples := []struct {
		input    *abci.TxResult
		expected string
		err      string
	}{
		{
			input:    &abci.TxResult{},
			expected: "DMLOG TX 0 0 CgIiAA==\n",
		},
		{
			input: &abci.TxResult{
				Index:  0,
				Height: 1000,
				Tx:     []byte("data"),
			},
			expected: "DMLOG TX 1000 0 CgsI6AcaBGRhdGEiAA==\n",
		},
	}

	for _, ex := range examples {
		output := bytes.NewBuffer(nil)
		extractor := NewExtractorService(nil, nil)
		extractor.writer = NewConsoleWriter(output)

		err := extractor.indexTX(ex.input)
		if err != nil {
			assert.Equal(t, err.Error(), ex.err)
		}
		assert.Equal(t, ex.expected, output.String())
	}
}

type mockSubscription struct {
	data     []pubsub.Message
	messages chan pubsub.Message
}

func (s mockSubscription) Out() <-chan pubsub.Message {
	go func() {
		for _, msg := range s.data {
			s.messages <- msg
		}
	}()
	return s.messages
}

func (s mockSubscription) Cancelled() <-chan struct{} {
	return nil
}

func (s mockSubscription) Err() error {
	return nil
}

func TestExtractorDrainSubscription(t *testing.T) {
	sub := mockSubscription{
		messages: make(chan pubsub.Message),
		data:     []pubsub.Message{{}, {}, {}, {}},
	}

	ex := NewExtractorService(nil, &Config{})
	err := ex.drainSubscription(sub, 3)
	assert.NoError(t, err)
}
