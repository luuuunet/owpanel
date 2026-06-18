package backup

// UploadToRemote uploads a local file to a configured remote backup target.
func (s *Service) UploadToRemote(remoteID uint, localFile, remoteName string) error {
	return s.uploadToRemote(remoteID, localFile, remoteName)
}
