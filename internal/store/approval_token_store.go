package store

import "sync"

var (
	approvalTokenMu  sync.RWMutex
	approvalTokenMap = make(map[string]ApprovalRequest)
)

func (s *Store) SetTokenMapping(tokenHash string, req ApprovalRequest) {
	approvalTokenMu.Lock()
	defer approvalTokenMu.Unlock()
	approvalTokenMap[tokenHash] = req
}

func (s *Store) GetTokenMapping(tokenHash string) (ApprovalRequest, bool) {
	approvalTokenMu.RLock()
	defer approvalTokenMu.RUnlock()
	req, ok := approvalTokenMap[tokenHash]
	return req, ok
}
