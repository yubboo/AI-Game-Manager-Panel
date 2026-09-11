package logcenter

import (
	"io"
)

type Service struct{ store *Store }

func NewService(store *Store) *Service { return &Service{store: store} }
func (s *Service) List(request ListRequest) ([]Session, error) {
	return s.store.List(request)
}
func (s *Service) Read(request ReadRequest) (ReadPage, error) {
	return s.store.Read(request)
}
func (s *Service) Tail(request TailRequest) (ReadPage, error) {
	return s.store.Tail(request)
}
func (s *Service) Search(request SearchRequest) (SearchResult, error) {
	return s.store.Search(request)
}
func (s *Service) Diagnostics(id string) (Diagnostics, error) {
	return s.store.Diagnostics(id)
}
func (s *Service) Export(id string) (ExportResult, error) { return s.store.Export(id) }
func (s *Service) File(id string) (FileRef, error)        { return s.store.File(id) }
func (s *Service) Bundle(request BundleRequest) (ExportResult, error) {
	return s.store.Bundle(request)
}

func (s *Service) BundleName(request BundleRequest) (string, error) {
	return s.store.BundleName(request)
}
func (s *Service) StreamBundle(request BundleRequest, writer io.Writer) (string, error) {
	return s.store.StreamBundle(request, writer)
}
