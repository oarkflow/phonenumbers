package phonenumbers

import (
	"fmt"
	"strings"

	"github.com/oarkflow/errors"
	"github.com/oarkflow/pkg/pool"
)

var Type = map[int]string{
	0: "FIXED_LINE",
	1: "MOBILE",
	// In some regions (e.g. the USA), it is impossible to distinguish between fixed-line and
	// mobile numbers by looking at the phone number itself.
	2: "FIXED_LINE_OR_MOBILE",
	// Freephone lines
	3: "TOLL_FREE",
	4: "PREMIUM_RATE",
	// The cost of this call is shared between the caller and the recipient, and is hence typically
	// less than PREMIUM_RATE calls. See // http://en.wikipedia.org/wiki/Shared_Cost_Service for
	// more information.
	5: "SHARED_COST",
	// Voice over IP numbers. This includes TSoIP (Telephony Service over IP).
	6: "VOIP",
	// A personal number is associated with a particular person, and may be routed to either a
	// MOBILE or FIXED_LINE number. Some more information can be found here:
	// http://en.wikipedia.org/wiki/Personal_Numbers
	7: "PERSONAL_NUMBER",
	8: "PAGER",
	// Used for "Universal Access Numbers" or "Company Numbers". They may be further routed to
	// specific offices, but allow one number to be used for a company.
	9: "UAN",
	// A phone number is of type UNKNOWN when it does not fit any of the known patterns for a
	// specific region.
	10: "UNKNOWN",
	// A phone number is of type UNKNOWN when it does not fit any of the known patterns for a
	// specific region.
	11: "UNKNOWN",

	// Emergency
	27: "EMERGENCY",

	// Voicemail
	28: "VOICEMAIL",

	// Short Code
	29: "SHORT_CODE",
}

type Number struct {
	Phone          string `query:"phone" json:"phone" csv:"phone"`
	DefaultPrefix  string `query:"default_prefix" json:"default_prefix" csv:"default_prefix"`
	PhoneTypeHuman string `json:"phone_type_human" csv:"phone_type_human"`
	CarrierName    string `json:"carrier_name" csv:"carrier_name"`
	CarrierMcc     string `json:"carrier_mcc" csv:"carrier_mcc"`
	CarrierMnc     string `json:"carrier_mnc" csv:"carrier_mnc"`
	CarrierNnc     string `json:"carrier_nnc" csv:"carrier_nnc"`
	CountryName    string `json:"country_name" csv:"country_name"`
	CountryCode    string `json:"country_code" csv:"country_code"`
	Currency       string `json:"currency" csv:"currency"`
	CurrencySymbol string `json:"currency_symbol" csv:"currency_symbol"`
	Timezone       string `json:"timezone" csv:"timezone"`
	Invalid        bool   `json:"invalid" csv:"invalid"`
	DialCode       int32  `json:"dial_code" csv:"dial_code"`
	PhoneType      int    `json:"phone_type" csv:"phone_type"`
}

type Numbers struct {
	Phones        []string `json:"phones"`
	DefaultPrefix string   `query:"default_prefix" json:"default_prefix"`
	PhoneTypes    []string `json:"phone_types"`
	PhoneOnly     bool     `json:"phone_only"`
}

type VerifiedNumbers struct {
	Phones        []Number `json:"phones"`
	DefaultPrefix string   `query:"default_prefix" json:"default_prefix"`
	PhoneTypes    []string `json:"phone_types"`
}

type UnverifiedNumbers struct {
	Phones []string `json:"phones"`
}

type Ops struct {
	Raw      string `json:"raw"`
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
	Alias    string `json:"alias"`
}

type Query struct {
	Raw        string   `json:"raw"`
	Fields     []string `json:"fields"`
	Operations []Ops    `json:"operations"`
	GroupBy    []string `json:"group_by"`
}

type AnalyzeCarrierResult struct {
	CarrierName string `json:"carrier_name"`
	CarrierCode string `json:"carrier_code"`
	CountryCode string `json:"country"`
	PhoneType   string `json:"phone_type"`
	PhoneCount  int    `json:"phone_count"`
}

type AnalyzeCountryResult struct {
	CountryCode string `json:"country"`
	PhoneType   string `json:"phone_type"`
	PhoneCount  int    `json:"phone_count"`
}

type CarrierStats struct {
	Result       map[string]*AnalyzeCarrierResult `json:"-"`
	RS           []*AnalyzeCarrierResult          `json:"result"`
	InvalidCount int                              `json:"invalid_count"`
	TotalCount   int                              `json:"total_count"`
}

type CountryStats struct {
	Result       map[string]*AnalyzeCountryResult `json:"-"`
	RS           []*AnalyzeCountryResult          `json:"result"`
	InvalidCount int                              `json:"invalid_count"`
	TotalCount   int                              `json:"total_count"`
}

var allowedPhoneTypes = map[string]int{
	"landline":           4,
	"mobile":             1,
	"mobile_or_landline": 2,
	"toll_free":          3,
	"voip":               6,
}

func (p *Number) Verify(getCarrierInfo ...bool) {
	carrierInfo := true
	if len(getCarrierInfo) > 0 && !getCarrierInfo[0] {
		carrierInfo = false
	}
	var num *PhoneNumber
	var e error
	p.Phone = strings.Trim(p.Phone, " ")
	if strings.HasPrefix(p.Phone, "00") {
		p.Phone = strings.Replace(p.Phone, "00", "+", 1)
	}
	hasPlusSymbol := strings.HasPrefix(p.Phone, "+")
	phoneWithoutPlus := strings.Replace(p.Phone, "+", "", 1)
	phoneWithPlus := "+" + phoneWithoutPlus
	if hasPlusSymbol {
		num, e = Parse(phoneWithPlus, "")
	} else {
		num, e = Parse(phoneWithoutPlus, strings.ToUpper(p.DefaultPrefix))
	}
	p.PhoneType = int(GetNumberType(num))
	if e == nil && (p.PhoneType == 10 || p.PhoneType == 11) {
		e = errors.New("unknown phone number")
	}
	if e != nil {
		num, e = Parse(phoneWithPlus, "")
	}
	if e != nil {
		num = nil
		p.Invalid = true
		return
	}
	p.PhoneType = int(GetNumberType(num))
	region := GetRegionCodeForNumber(num)
	p.CountryCode = region
	p.DefaultPrefix = region
	p.PhoneTypeHuman = Type[p.PhoneType]
	if p.PhoneType == 10 || p.PhoneType == 11 {
		num = nil
		p.Invalid = true
		return
	}

	p.Phone = Format(num, E164)
	timezones, e := GetTimezonesForNumber(num)
	if e != nil {
		num = nil
		p.Invalid = true
		return
	}
	if len(timezones) > 0 {
		p.Timezone = timezones[0]
	}
	p.DialCode = num.GetCountryCode()

	country, _ := Countries[region]
	p.CountryName = country.Name
	p.Currency = country.Currency
	p.CurrencySymbol = country.CurrencySymbol
	if carrierInfo {
		carrier, _ := GetCarrierForNumber(num, "EN")
		p.CarrierName = carrier
		networks := CountryNetwork[region]
		for _, net := range networks {
			if carrier != "" && (strings.Contains(strings.ToLower(net.Brand), strings.ToLower(carrier)) ||
				strings.Contains(strings.ToLower(net.Operator), strings.ToLower(carrier))) {
				p.CarrierMnc = net.Mnc
				p.CarrierMcc = net.Mcc
				p.CarrierNnc = net.Mcc + net.Mnc
				break
			}
		}
	}
}

func (p *Numbers) Verify() VerifiedNumbers {
	numbers := VerifiedNumbers{}
	workerPool := pool.New()
	defer workerPool.Close()

	batch := workerPool.Batch()

	go func() {
		for _, phone := range p.Phones {
			num := Number{Phone: phone, DefaultPrefix: p.DefaultPrefix}
			batch.Queue(verify(num))
		}
		batch.QueueComplete()
	}()
	for phone := range batch.Results() {
		numbers.Phones = append(numbers.Phones, phone.Value().(Number))
	}
	return numbers
}

func (p *Numbers) Clean() (VerifiedNumbers, UnverifiedNumbers) {
	phones := UnverifiedNumbers{}
	numbers := VerifiedNumbers{DefaultPrefix: p.DefaultPrefix, PhoneTypes: p.PhoneTypes}
	workerPool := pool.New()
	defer workerPool.Close()

	batch := workerPool.Batch()

	go func() {
		for _, phone := range p.Phones {
			num := Number{Phone: phone, DefaultPrefix: p.DefaultPrefix}
			batch.Queue(verify(num))
		}
		batch.QueueComplete()
	}()
	for phone := range batch.Results() {
		num := phone.Value().(Number)
		if !num.Invalid {
			if len(p.PhoneTypes) > 0 {
				for _, phoneType := range p.PhoneTypes {
					allowed := allowedPhoneTypes[strings.ToLower(phoneType)]
					if allowed == 4 && num.PhoneType == 0 {
						if p.PhoneOnly {
							phones.Phones = append(phones.Phones, num.Phone)
						} else {
							numbers.Phones = append(numbers.Phones, num)
						}

					} else if allowed > 0 && num.PhoneType == allowed {
						if p.PhoneOnly {
							phones.Phones = append(phones.Phones, num.Phone)
						} else {
							numbers.Phones = append(numbers.Phones, num)
						}
					}
				}
			} else {
				if p.PhoneOnly {
					phones.Phones = append(phones.Phones, num.Phone)
				} else {
					numbers.Phones = append(numbers.Phones, num)
				}
			}

		}
	}
	return numbers, phones
}

func (p *Numbers) StatsByCarrier() CarrierStats {
	stats := CarrierStats{}
	rs := make(map[string]*AnalyzeCarrierResult)
	workerPool := pool.New()
	defer workerPool.Close()

	batch := workerPool.Batch()
	stats.TotalCount = len(p.Phones)
	go func() {
		for _, phone := range p.Phones {
			num := Number{Phone: phone, DefaultPrefix: p.DefaultPrefix}
			batch.Queue(verify(num))
		}
		batch.QueueComplete()
	}()
	for phone := range batch.Results() {
		num := phone.Value().(Number)
		if num.Invalid {
			stats.InvalidCount += 1
			continue
		}
		hash := num.CountryCode
		if num.CarrierNnc != "" {
			hash = hash + ":" + num.CarrierNnc
		}
		hash = hash + ":" + num.PhoneTypeHuman
		analyzeResult := &AnalyzeCarrierResult{
			CarrierName: num.CarrierName,
			CarrierCode: num.CarrierNnc,
			CountryCode: num.CountryCode,
			PhoneType:   num.PhoneTypeHuman,
		}

		if rs[hash] == nil {
			analyzeResult.PhoneCount = 1
			rs[hash] = analyzeResult
		} else {
			rs[hash].PhoneCount += 1
		}
	}

	for _, val := range rs {
		stats.RS = append(stats.RS, val)
	}
	rs = nil
	return stats
}

func (p *Numbers) StatsByCountry() CountryStats {
	stats := CountryStats{}
	rs := make(map[string]*AnalyzeCountryResult)
	workerPool := pool.New()
	defer workerPool.Close()

	batch := workerPool.Batch()
	stats.TotalCount = len(p.Phones)
	go func() {
		for _, phone := range p.Phones {
			num := Number{Phone: phone, DefaultPrefix: p.DefaultPrefix}
			batch.Queue(verify(num))
		}
		batch.QueueComplete()
	}()
	for phone := range batch.Results() {
		num := phone.Value().(Number)
		if num.Invalid {
			stats.InvalidCount += 1
			continue
		}
		hash := num.CountryCode
		hash = hash + ":" + num.PhoneTypeHuman
		analyzeResult := &AnalyzeCountryResult{
			CountryCode: num.CountryCode,
			PhoneType:   num.PhoneTypeHuman,
		}

		if rs[hash] == nil {
			analyzeResult.PhoneCount = 1
			rs[hash] = analyzeResult
		} else {
			rs[hash].PhoneCount += 1
		}
	}
	for _, val := range rs {
		stats.RS = append(stats.RS, val)
	}
	rs = nil
	return stats
}

func verify(phone Number) pool.WorkFunc {
	return func(wu pool.WorkUnit) (interface{}, error) {
		if wu.IsCancelled() {
			return nil, nil
		}
		phone.Verify()
		return phone, nil
	}
}

func Verify(phone string, defaultRegion ...string) Number {
	defaultPrefix := ""
	if len(defaultRegion) > 0 {
		defaultPrefix = defaultRegion[0]
	}
	num := Number{
		Phone:         phone,
		DefaultPrefix: defaultPrefix,
	}
	num.Verify(true)
	return num
}

func Clean(phone []string, defaultRegion ...string) (VerifiedNumbers, UnverifiedNumbers) {
	defaultPrefix := ""
	if len(defaultRegion) > 0 {
		defaultPrefix = defaultRegion[0]
	}
	nums := Numbers{
		Phones:        phone,
		DefaultPrefix: defaultPrefix,
	}
	return nums.Clean()
}

func VerifyList(phone []string, defaultRegion ...string) VerifiedNumbers {
	defaultPrefix := ""
	if len(defaultRegion) > 0 {
		defaultPrefix = defaultRegion[0]
	}
	nums := Numbers{
		Phones:        phone,
		DefaultPrefix: defaultPrefix,
	}
	return nums.Verify()
}

func StatsByCarrier(phone []string, defaultRegion ...string) CarrierStats {
	defaultPrefix := ""
	if len(defaultRegion) > 0 {
		defaultPrefix = defaultRegion[0]
	}
	nums := Numbers{
		Phones:        phone,
		DefaultPrefix: defaultPrefix,
	}
	return nums.StatsByCarrier()
}

func StatsByCountry(phone []string, defaultRegion ...string) CountryStats {
	defaultPrefix := ""
	if len(defaultRegion) > 0 {
		defaultPrefix = defaultRegion[0]
	}
	nums := Numbers{
		Phones:        phone,
		DefaultPrefix: defaultPrefix,
	}
	return nums.StatsByCountry()
}

func Worker(data map[string]string) bool {
	var num Number
	num.Phone = data["phone"]
	num.Verify()
	if num.Invalid {
		return false
	}
	data["phone"] = num.Phone
	data["phone_type_code"] = fmt.Sprintf("%v", num.PhoneType)
	data["phone_type_label"] = num.PhoneTypeHuman
	data["carrier_name"] = num.CarrierName
	data["carrier_mnc"] = num.CarrierMnc
	data["carrier_mcc"] = num.CarrierMcc
	data["carrier_nnc"] = num.CarrierNnc
	data["dial_code"] = fmt.Sprintf("%v", num.DialCode)
	data["region"] = num.CountryCode
	data["currency"] = num.Currency
	data["currency_symbol"] = num.CurrencySymbol
	return true
}
