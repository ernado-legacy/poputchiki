package main

import (
	"errors"
	"github.com/ginuerzh/weedo"
	"strconv"
	"strings"
)

var (
	InvalidFid = errors.New("invalid fid")
)

type WeedAdapter struct {
	client  *weedo.Client
	volumes map[int]string
}

func NewAdapter() *WeedAdapter {
	w := &WeedAdapter{}
	w.volumes = make(map[int]string)
	w.client = weedo.NewClient(weedHost, weedPort)
	return w
}

func (adapter *WeedAdapter) GetUrl(fid string) (url string, err error) {
	if len(fid) < 5 {
		return "", InvalidFid
	}
	index := strings.Index(fid, ",")
	if index == -1 && index == 0 {
		return "", InvalidFid
	}
	volumeId, err := strconv.Atoi(fid[:index])
	if err != nil {
		return "", InvalidFid
	}

	volumeUrl, ok := adapter.volumes[volumeId]

	if !ok {
		volumeUrl, _, err = adapter.client.Lookup(uint64(volumeId))
		if err != nil {
			return
		}
		adapter.volumes[volumeId] = volumeUrl
	}
	url = volumeUrl + "/" + fid
	return
}
