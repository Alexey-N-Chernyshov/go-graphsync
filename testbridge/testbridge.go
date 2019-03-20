package testbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	cid "github.com/ipfs/go-cid"
	ipldbridge "github.com/ipfs/go-graphsync/ipldbridge"
	ipld "github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/linking/cid"
	multihash "github.com/multiformats/go-multihash"
)

type mockIPLDBridge struct {
}

// NewMockIPLDBridge returns an IPLD bridge that works with MockSelectors
func NewMockIPLDBridge() ipldbridge.IPLDBridge {
	return &mockIPLDBridge{}
}

func (mb *mockIPLDBridge) ValidateSelectorSpec(cidRootedSelector ipld.Node) []error {
	spec, ok := cidRootedSelector.(*mockSelectorSpec)
	if !ok || spec.failValidation {
		return []error{fmt.Errorf("not a selector")}
	}
	return nil
}

func (mb *mockIPLDBridge) EncodeNode(node ipld.Node) ([]byte, error) {
	spec, ok := node.(*mockSelectorSpec)
	if ok && !spec.failEncode {
		data, err := json.Marshal(spec.cidsVisited)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	return nil, fmt.Errorf("format not supported")
}

func (mb *mockIPLDBridge) DecodeNode(data []byte) (ipld.Node, error) {
	var cidsVisited []cid.Cid
	err := json.Unmarshal(data, &cidsVisited)
	if err == nil {
		return &mockSelectorSpec{cidsVisited, false, false}, nil
	}
	return nil, fmt.Errorf("format not supported")
}

func (mb *mockIPLDBridge) DecodeSelectorSpec(cidRootedSelector ipld.Node) (ipld.Node, ipldbridge.Selector, error) {
	spec, ok := cidRootedSelector.(*mockSelectorSpec)
	if !ok || spec.failValidation {
		return nil, nil, fmt.Errorf("not a selector")
	}
	return nil, newMockSelector(spec), nil
}

func (mb *mockIPLDBridge) Traverse(ctx context.Context, loader ipldbridge.RawLoader, root ipld.Node, s ipldbridge.Selector, fn ipldbridge.AdvVisitFn) error {
	ms, ok := s.(*mockSelector)
	if !ok {
		return fmt.Errorf("not supported")
	}
	var lastErr error
	for _, lnk := range ms.cidsVisited {

		node, err := loadNode(lnk, loader)
		if err != nil {
			lastErr = err
		} else {
			fn(ipldbridge.TraversalProgress{}, node, 0)
		}
	}
	return lastErr
}

func loadNode(lnk cid.Cid, loader ipldbridge.RawLoader) (ipld.Node, error) {
	r, err := loader(cidlink.Link{Cid: lnk}, ipldbridge.LinkContext{})
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	io.Copy(&buffer, r)
	data := buffer.Bytes()
	hash, err := multihash.Sum(data, lnk.Prefix().MhType, lnk.Prefix().MhLength)
	if err != nil {
		return nil, err
	}
	if hash.B58String() != lnk.Hash().B58String() {
		return nil, fmt.Errorf("hash mismatch")
	}
	return NewMockBlockNode(data), nil
}
