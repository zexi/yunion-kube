package cloudprovider

import (
	"context"
	"fmt"

	"errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	ErrNoSuchProvder = errors.New("no such provider")
)

type SCloudaccount struct {
	Account string
	Secret  string
}

type ICloudProviderFactory interface {
	GetProvider(providerId, providerName, url, account, secret string) (ICloudProvider, error)
	GetId() string
	ValidateChangeBandwidth(instanceId string, bandwidth int64) error
	ValidateCreateCloudaccountData(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) error
	ValidateUpdateCloudaccountCredential(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, cloudaccount string) (*SCloudaccount, error)
}

type ICloudProvider interface {
	GetId() string
	GetName() string
	GetSysInfo() (jsonutils.JSONObject, error)
	GetVersion() string
	IsPublicCloud() bool
	IsOnPremiseInfrastructure() bool

	GetIRegions() []ICloudRegion
	GetIRegionById(id string) (ICloudRegion, error)

	GetOnPremiseIRegion() (ICloudRegion, error)

	// GetIHostById(id string) (ICloudHost, error)
	// GetIVpcById(id string) (ICloudVpc, error)
	// GetIStorageById(id string) (ICloudStorage, error)
	// GetIStoragecacheById(id string) (ICloudStoragecache, error)

	GetBalance() (float64, error)

	GetSubAccounts() ([]SSubAccount, error)

	SupportPrepaidResources() bool
}

var providerTable map[string]ICloudProviderFactory

func init() {
	providerTable = make(map[string]ICloudProviderFactory)
}

func RegisterFactory(factory ICloudProviderFactory) {
	providerTable[factory.GetId()] = factory
}

func GetProviderDriver(provider string) (ICloudProviderFactory, error) {
	factory, ok := providerTable[provider]
	if ok {
		return factory, nil
	}
	log.Errorf("Provider %s not registerd", provider)
	return nil, fmt.Errorf("No such provider %s", provider)
}

func GetRegistedProviderIds() []string {
	providers := []string{}
	for id := range providerTable {
		providers = append(providers, id)
	}

	return providers
}

func GetProvider(providerId, providerName, accessUrl, account, secret, provider string) (ICloudProvider, error) {
	driver, err := GetProviderDriver(provider)
	if err != nil {
		return nil, err
	}
	return driver.GetProvider(providerId, providerName, accessUrl, account, secret)
}

func IsSupported(provider string) bool {
	_, ok := providerTable[provider]
	return ok
}

func IsValidCloudAccount(accessUrl, account, secret, provider string) error {
	factory, ok := providerTable[provider]
	if ok {
		_, err := factory.GetProvider("", "", accessUrl, account, secret)
		return err
	} else {
		return ErrNoSuchProvder
	}
}
