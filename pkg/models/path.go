package models

import (
	"errors"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

type ResultPath struct {
	// Name
	Name string `json:"Name"`
	// The path to the file/dir
	Path string `json:"Path"`
}

// Normalize normalizes the path to a canonical form
func (p *ResultPath) Normalize() {
	if p == nil {
		return
	}
	p.Name = strings.TrimSpace(p.Name)
	p.Path = strings.TrimSpace(p.Path)
}

// Copy returns a copy of the path
func (p *ResultPath) Copy() *ResultPath {
	if p == nil {
		return nil
	}
	return &ResultPath{
		Name: p.Name,
		Path: p.Path,
	}
}

// Validate validates the path
func (p *ResultPath) Validate() error {
	if p == nil {
		return errors.New("path is nil")
	}
	var mErr error
	if validate.IsBlank(p.Path) {
		mErr = errors.Join(mErr, errors.New("path is blank"))
	}
	if validate.IsBlank(p.Name) {
		mErr = errors.Join(mErr, errors.New("resultpath name is blank"))
	}
	return mErr
}
