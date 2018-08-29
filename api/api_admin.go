package api

// PrivateBizAPI provides an API to business managed by this server.

type PrivateAdminAPI struct {
	w *accounts.Manager
	b Backend
}

// NewPrivateAdminAPI create a new NewPrivateAdminAPI.
func NewPrivateAdminAPI(b Backend) *PrivateAdminAPI {
	return &PrivateAdminAPI{
		w: b.WalletManager(),
		b: b,
	}
}

// ListAccounts will return a list of addresses for accounts this node manages.
func (s *PrivateAdminAPI) Accounts() ([]common.Address, error) {

	return paccounts.NewWalletWrapper(s.w.Wallets()).Accounts()
}
