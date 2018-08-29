package api

const (
	defaultGasPrice = 50
)

// PublicBizAPI provides an API to access accounts managed by this node.
// It offers only methods that can retrieve accounts.
type PublicBizAPI struct {
	w string
}

// NewPublicAccountAPI creates a new PublicBizAPI.
func NewPublicBizAPI(aw *paccounts.WalletManager) *PublicBizAPI {
	return &PublicBizAPI{w: paccounts.GetWalletManager()}
}

// Accounts returns the collection of accounts this node manages
func (s *PublicBizAPI) Accounts() ([]common.Address, error) {

	return paccounts.NewWalletWrapper(s.w.Wallets()).Accounts()
}
