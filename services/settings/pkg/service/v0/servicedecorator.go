package svc

import (
	"context"
	"net/http"
	"strings"

	settingsmsg "github.com/owncloud/ocis/v2/protogen/gen/ocis/messages/settings/v0"
	settingssvc "github.com/owncloud/ocis/v2/protogen/gen/ocis/services/settings/v0"
	"github.com/owncloud/ocis/v2/services/settings/pkg/config"
	"github.com/owncloud/ocis/v2/services/settings/pkg/settings"
	"github.com/owncloud/ocis/v2/services/settings/pkg/store/defaults"
	merrors "go-micro.dev/v4/errors"
)

var _defaultLanguage = "en"

// NewDefaultLanguageService returns a default language decorator for ServiceHandler.
func NewDefaultLanguageService(cfg *config.Config, serviceHandler settings.ServiceHandler) settings.ServiceHandler {
	defaultLanguage := cfg.DefaultLanguage
	if defaultLanguage == "" {
		defaultLanguage = _defaultLanguage
	}
	return &defaultLanguageDecorator{defaultLanguage: defaultLanguage, ServiceHandler: serviceHandler}
}

type defaultLanguageDecorator struct {
	defaultLanguage string
	settings.ServiceHandler
}

// GetValueByUniqueIdentifiers implements the ValueService interface
func (s *defaultLanguageDecorator) GetValueByUniqueIdentifiers(ctx context.Context, req *settingssvc.GetValueByUniqueIdentifiersRequest, res *settingssvc.GetValueResponse) error {
	err := s.ServiceHandler.GetValueByUniqueIdentifiers(ctx, req, res)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") && res.GetValue() == nil {
			defaultValueList := getDefaultValueList()
			// Ensure the default values for profile settings
			if _, ok := defaultValueList[req.GetSettingId()]; ok {
				res.Value = s.withDefaultProfileValue(ctx, req.AccountUuid, req.GetSettingId())
				return nil
			}
		}
		return err
	}
	return nil
}

// ListValues implements the ValueServiceHandler interface
func (s *defaultLanguageDecorator) ListValues(ctx context.Context, req *settingssvc.ListValuesRequest, res *settingssvc.ListValuesResponse) error {
	err := s.ServiceHandler.ListValues(ctx, req, res)
	if err != nil {
		return err
	}
	defaultValueList := getDefaultValueList()
	for _, v := range res.Values {
		if _, ok := defaultValueList[v.GetValue().GetSettingId()]; ok {
			delete(defaultValueList, v.GetValue().GetSettingId())
		}
	}

	// Ensure the default values for profile settings
	defaultValueList = s.withDefaultProfileValueList(ctx, req.AccountUuid, defaultValueList)
	if len(defaultValueList) > 0 {
		for _, v := range defaultValueList {
			if v != nil {
				res.Values = append(res.Values, v)
			}
		}
	}

	return nil
}

func (s *defaultLanguageDecorator) withDefaultLanguageSetting(accountUUID string) *settingsmsg.ValueWithIdentifier {
	return &settingsmsg.ValueWithIdentifier{
		Identifier: &settingsmsg.Identifier{
			Extension: "ocis-accounts",
			Bundle:    "profile",
			Setting:   "language",
		},
		Value: &settingsmsg.Value{
			BundleId:    defaults.BundleUUIDProfile,
			SettingId:   defaults.SettingUUIDProfileLanguage,
			AccountUuid: accountUUID,
			Resource: &settingsmsg.Resource{
				Type: settingsmsg.Resource_TYPE_USER,
			},
			Value: &settingsmsg.Value_ListValue{
				ListValue: &settingsmsg.ListValue{Values: []*settingsmsg.ListOptionValue{
					{
						Option: &settingsmsg.ListOptionValue_StringValue{
							StringValue: s.defaultLanguage,
						},
					},
				}},
			},
		},
	}
}

func (s *defaultLanguageDecorator) withDefaultProfileValue(ctx context.Context, accountUUID string, settingId string) *settingsmsg.ValueWithIdentifier {
	if settingId == defaults.SettingUUIDProfileLanguage {
		return s.withDefaultLanguageSetting(accountUUID)
	}
	res := s.withDefaultProfileValueList(ctx, accountUUID, map[string]*settingsmsg.ValueWithIdentifier{settingId: nil})
	if v, ok := res[settingId]; ok {
		return v
	}
	return nil
}

func (s *defaultLanguageDecorator) withDefaultProfileValueList(ctx context.Context,
	accountUUID string, requested map[string]*settingsmsg.ValueWithIdentifier) map[string]*settingsmsg.ValueWithIdentifier {
	req := &settingssvc.GetBundleRequest{BundleId: defaults.BundleUUIDProfile}
	resp := &settingssvc.GetBundleResponse{}
	if err := s.GetBundle(ctx, req, resp); err != nil {
		if merr, ok := merrors.As(err); ok && merr.Code != http.StatusNotFound {
			return requested
		}
	}

	for _, setting := range resp.GetBundle().GetSettings() {
		if v, ok := requested[setting.GetId()]; ok && v == nil {
			if setting.GetId() == defaults.SettingUUIDProfileLanguage {
				requested[setting.GetId()] = s.withDefaultLanguageSetting(accountUUID)
				continue
			}

			newVal := &settingsmsg.ValueWithIdentifier{
				Identifier: &settingsmsg.Identifier{
					Extension: resp.GetBundle().GetExtension(),
					Bundle:    resp.GetBundle().GetName(),
					Setting:   setting.GetName(),
				},
				Value: &settingsmsg.Value{
					BundleId:    resp.GetBundle().GetId(),
					SettingId:   setting.GetId(),
					AccountUuid: accountUUID,
					Resource:    setting.GetResource(),
				},
			}

			switch val := setting.GetValue().(type) {
			case *settingsmsg.Setting_MultiChoiceCollectionValue:
				newVal.Value.Value = multiChoiceCollectionToValue(val.MultiChoiceCollectionValue)
				requested[setting.GetId()] = newVal
			}
		}
	}

	return requested
}

func multiChoiceCollectionToValue(collection *settingsmsg.MultiChoiceCollection) *settingsmsg.Value_CollectionValue {
	values := make([]*settingsmsg.CollectionOption, 0, len(collection.GetOptions()))
	for _, option := range collection.GetOptions() {
		switch o := option.GetValue().GetOption().(type) {
		case *settingsmsg.MultiChoiceCollectionOptionValue_BoolValue:
			if o != nil {
				values = append(values, &settingsmsg.CollectionOption{
					Key: option.GetKey(),
					Option: &settingsmsg.CollectionOption_BoolValue{
						BoolValue: o.BoolValue.GetDefault(),
					},
				})
			}
		}
	}

	return &settingsmsg.Value_CollectionValue{
		CollectionValue: &settingsmsg.CollectionValue{
			Values: values,
		},
	}
}

func getDefaultValueList() map[string]*settingsmsg.ValueWithIdentifier {
	return map[string]*settingsmsg.ValueWithIdentifier{
		defaults.SettingUUIDProfileLanguage:                             nil,
		defaults.SettingUUIDProfileEventShareCreated:                    nil,
		defaults.SettingUUIDProfileEventShareRemoved:                    nil,
		defaults.SettingUUIDProfileEventShareExpired:                    nil,
		defaults.SettingUUIDProfileEventSpaceShared:                     nil,
		defaults.SettingUUIDProfileEventSpaceUnshared:                   nil,
		defaults.SettingUUIDProfileEventSpaceMembershipExpired:          nil,
		defaults.SettingUUIDProfileEventSpaceDisabled:                   nil,
		defaults.SettingUUIDProfileEventSpaceDeleted:                    nil,
		defaults.SettingUUIDProfileEventPostprocessingStepFinished:      nil,
		defaults.SettingUUIDProfileEventScienceMeshInviteTokenGenerated: nil,
	}
}
