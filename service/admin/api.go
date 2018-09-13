package admin

import ts "airman.com/airtask/pkg/types"

// PrivateBizAPI provides an API to business managed by this server.
type PrivateAdminAPI struct {
	//w *admin.Manager
	b ts.Backend
}

// NewPrivateAdminAPI create a new NewPrivateAdminAPI.
func NewPrivateAdminAPI(b ts.Backend) *PrivateAdminAPI {
	return &PrivateAdminAPI{
		//w: b.WalletManager(),
		b: b,
	}
}

// ListAccounts will return a list of addresses for accounts this node manages.
func (s *PrivateAdminAPI) Accounts() error {

	//return paccounts.NewWalletWrapper(s.w.Wallets()).Accounts()
	return nil
}
