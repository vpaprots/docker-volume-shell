package main

import (
	"fmt"

	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
)

func (p *volumePlugin) Create(req *volume.CreateRequest) error {
	logCtx := log.WithFields(
		log.Fields{
			"function": "Volume.Create",
			"name":     req.Name,
		},
	)
	logCtx.Infof("request: %+v", req)
	req.Options["name"] = req.Name

	for k, vDefault := range p.config.Defaults {
		if _, ok := req.Options[k]; !ok {
			req.Options[k] = vDefault
		}
	}

	_, err := templatedCmd(logCtx, p.config.Templates.Create, req.Options)
	return err
}

func (p *volumePlugin) Remove(req *volume.RemoveRequest) error {
	logCtx := log.WithFields(
		log.Fields{
			"function": "Volume.Remove",
			"name":     req.Name,
		},
	)
	logCtx.Infof("request: %+v", req)
	templateFill := map[string]string{"name": req.Name}

	_, err := templatedCmd(logCtx, p.config.Templates.Remove, templateFill)
	return err
	//return fmt.Errorf("no such volume")
}

func (p *volumePlugin) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	logCtx := log.WithFields(
		log.Fields{
			"function": "Volume.Mount",
			"name":     req.Name,
		},
	)
	logCtx.Infof("request: %+v", req)
	templateFill := map[string]string{"name": req.Name}

	cmdOut, err := templatedCmd(logCtx, p.config.Templates.Mount, templateFill)
	if err != nil {
		return &volume.MountResponse{}, err
	}
	return &volume.MountResponse{
		Mountpoint: cmdOut[len(cmdOut)-1],
	}, nil
}

func (p *volumePlugin) Unmount(req *volume.UnmountRequest) error {
	logCtx := log.WithFields(
		log.Fields{
			"function": "Volume.Unmount",
			"name":     req.Name,
		},
	)
	logCtx.Infof("request: %+v", req)
	templateFill := map[string]string{"name": req.Name}

	_, err := templatedCmd(logCtx, p.config.Templates.Unmount, templateFill)
	return err
}

func (p *volumePlugin) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	logCtx := log.WithFields(
		log.Fields{
			"function": "Volume.Get",
			"name":     req.Name,
		},
	)
	logCtx.Infof("request: %+v", req)
	templateFill := map[string]string{"name": req.Name}

	_, err := templatedCmd(logCtx, p.config.Templates.Get, templateFill)
	if err != nil {
		return &volume.GetResponse{}, fmt.Errorf("no such volume")
	}
	return &volume.GetResponse{Volume: &volume.Volume{Name: req.Name}}, nil
}

func (p *volumePlugin) List() (*volume.ListResponse, error) {
	logCtx := log.WithFields(
		log.Fields{
			"function": "Volume.List",
		},
	)
	logCtx.Infof("called")
	templateFill := map[string]string{}

	list, err := templatedCmd(logCtx, p.config.Templates.List, templateFill)
	if err != nil {
		return nil, err
	}

	var vols []*volume.Volume
	for _, v := range list {
		vols = append(vols, &volume.Volume{Name: v})
	}
	return &volume.ListResponse{Volumes: vols}, nil
}

func (p *volumePlugin) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	logCtx := log.WithFields(
		log.Fields{
			"function": "Volume.Path",
			"name":     req.Name,
		},
	)
	logCtx.Infof("request: %+v", req)
	templateFill := map[string]string{"name": req.Name}

	path, err := templatedCmd(logCtx, p.config.Templates.Path, templateFill)
	if err != nil {
		return nil, err
	}

	return &volume.PathResponse{
		Mountpoint: path[len(path)-1],
	}, nil
}

func (p *volumePlugin) Capabilities() *volume.CapabilitiesResponse {
	log.Infof("volumePlugin.Capabilities called")
	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "global"}}
}
