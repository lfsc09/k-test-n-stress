package mock

import (
	"fmt"
	"strconv"

	"github.com/lfsc09/kmock"
)

type Faker struct {
	kmock *kmock.KMock
}

func NewFaker() *Faker {
	return &Faker{
		kmock: kmock.New(),
	}
}

// Generate generates a fake value based on the provided function name and arguments.
// It returns the generated value as a string or an error if the function name is unknown or if there are issues with the arguments.
func (f Faker) Generate(fn string, args []string) (string, error) {
	switch fn {
	/*
		ADDRESSES
	*/
	case "Address.Country":
		return f.kmock.Address.Country(), nil

	case "Address.CountryCode":
		return f.kmock.Address.CountryCode(), nil

	case "Address.State":
		if len(args) > 1 {
			return "", fmt.Errorf("Address.State: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		state, err := f.kmock.Address.State(localeParam)
		if err != nil {
			return "", fmt.Errorf("Address.State: %v", err)
		}
		return state, nil

	case "Address.StateCode":
		if len(args) > 1 {
			return "", fmt.Errorf("Address.StateCode: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		stateCode, err := f.kmock.Address.StateCode(localeParam)
		if err != nil {
			return "", fmt.Errorf("Address.StateCode: %v", err)
		}
		return stateCode, nil

	case "Address.City":
		if len(args) > 1 {
			return "", fmt.Errorf("Address.City: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		city, err := f.kmock.Address.City(localeParam)
		if err != nil {
			return "", fmt.Errorf("Address.City: %v", err)
		}
		return city, nil

	case "Address.CityFromState":
		if len(args) > 2 {
			return "", fmt.Errorf("Address.CityFromState: expected 0, 1 or 2 arguments, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		stateCodeParam := argOr(args, 1, "")
		city, err := f.kmock.Address.CityFromState(localeParam, stateCodeParam)
		if err != nil {
			return "", fmt.Errorf("Address.CityFromState: %v", err)
		}
		return city, nil

	case "Address.Neighborhood":
		if len(args) > 1 {
			return "", fmt.Errorf("Address.Neighborhood: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		neighborhood, err := f.kmock.Address.Neighborhood(localeParam)
		if err != nil {
			return "", fmt.Errorf("Address.Neighborhood: %v", err)
		}
		return neighborhood, nil

	case "Address.StreetName":
		if len(args) > 1 {
			return "", fmt.Errorf("Address.StreetName: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		streetName, err := f.kmock.Address.StreetName(localeParam)
		if err != nil {
			return "", fmt.Errorf("Address.StreetName: %v", err)
		}
		return streetName, nil

	case "Address.StreetNumber":
		return f.kmock.Address.StreetNumber(), nil

	case "Address.StreetComplement":
		if len(args) > 1 {
			return "", fmt.Errorf("Address.StreetComplement: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		streetComplement, err := f.kmock.Address.StreetComplement(localeParam)
		if err != nil {
			return "", fmt.Errorf("Address.StreetComplement: %v", err)
		}
		return streetComplement, nil

	case "Address.ZipCode":
		if len(args) > 1 {
			return "", fmt.Errorf("Address.ZipCode: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		zipCode, err := f.kmock.Address.ZipCode(localeParam)
		if err != nil {
			return "", fmt.Errorf("Address.ZipCode: %v", err)
		}
		return zipCode, nil

	case "Address.Latitude":
		return strconv.FormatFloat(f.kmock.Address.Latitude(), 'f', 6, 64), nil

	case "Address.Longitude":
		return strconv.FormatFloat(f.kmock.Address.Longitude(), 'f', 6, 64), nil

	/*
		BOOLEANS
	*/
	case "Boolean.Random":
		return strconv.FormatBool(f.kmock.Boolean.Random()), nil

	case "Boolean.RandomWithProbability":
		if len(args) != 1 {
			return "", fmt.Errorf("Boolean.RandomWithProbability: expected 1 argument, got %d", len(args))
		}
		probability, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return "", fmt.Errorf("Boolean.RandomWithProbability: invalid probability value '%s'", args[0])
		}
		return strconv.FormatBool(f.kmock.Boolean.RandomWithProbability(probability)), nil

	/*
		CARS
	*/
	case "Car.Brand":
		return f.kmock.Car.Brand(), nil

	case "Car.Model":
		return f.kmock.Car.Model(), nil

	case "Car.LicensePlate":
		if len(args) > 1 {
			return "", fmt.Errorf("Car.LicensePlate: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		licensePlate, err := f.kmock.Car.LicensePlate(localeParam)
		if err != nil {
			return "", fmt.Errorf("Car.LicensePlate: %v", err)
		}
		return licensePlate, nil

	case "Car.Color":
		return f.kmock.Car.Color(), nil

	/*
		COMPANIES
	*/
	case "Company.Name":
		if len(args) > 1 {
			return "", fmt.Errorf("Company.Name: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		name, err := f.kmock.Company.Name(localeParam)
		if err != nil {
			return "", fmt.Errorf("Company.Name: %v", err)
		}
		return name, nil

	case "Company.Dba":
		if len(args) > 1 {
			return "", fmt.Errorf("Company.Dba: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		dba, err := f.kmock.Company.Dba(localeParam)
		if err != nil {
			return "", fmt.Errorf("Company.Dba: %v", err)
		}
		return dba, nil

	case "Company.Industry":
		if len(args) > 1 {
			return "", fmt.Errorf("Company.Industry: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		industry, err := f.kmock.Company.Industry(localeParam)
		if err != nil {
			return "", fmt.Errorf("Company.Industry: %v", err)
		}
		return industry, nil

	case "Company.Suffix":
		if len(args) > 1 {
			return "", fmt.Errorf("Company.Suffix: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		suffix, err := f.kmock.Company.Suffix(localeParam)
		if err != nil {
			return "", fmt.Errorf("Company.Suffix: %v", err)
		}
		return suffix, nil

	case "Company.EIN":
		return f.kmock.Company.EIN(), nil

	case "Company.CNPJLegacyValid":
		return f.kmock.Company.CNPJLegacyValid(), nil

	case "Company.CNPJLegacyInvalid":
		return f.kmock.Company.CNPJLegacyInvalid(), nil

	case "Company.CNPJAlphanumericValid":
		return f.kmock.Company.CNPJAlphanumericValid(), nil

	case "Company.CNPJAlphanumericInvalid":
		return f.kmock.Company.CNPJAlphanumericInvalid(), nil

	case "Company.IE":
		return f.kmock.Company.IE(), nil

	case "Company.CNAE":
		return f.kmock.Company.CNAE(), nil

	/*
		CURRENCIES
	*/
	case "Currency.Name":
		return f.kmock.Currency.Name(), nil

	case "Currency.Code":
		return f.kmock.Currency.Code(), nil

	case "Currency.Symbol":
		return f.kmock.Currency.Symbol(), nil

	/*
		DATES
	*/
	case "Date.Date":
		if len(args) > 3 {
			return "", fmt.Errorf("Date.Date: expected 0 to 3 arguments, got %d", len(args))
		}
		fromParam := argOr(args, 0, "")
		toParam := argOr(args, 1, "")
		formatParam := argOr(args, 2, "")
		return f.kmock.Date.Date(fromParam, toParam, formatParam), nil

	case "Date.Time":
		if len(args) > 3 {
			return "", fmt.Errorf("Date.Time: expected 0 to 3 arguments, got %d", len(args))
		}
		fromParam := argOr(args, 0, kmock.EN_US)
		toParam := argOr(args, 1, "")
		formatParam := argOr(args, 2, "")
		return f.kmock.Date.Time(fromParam, toParam, formatParam), nil

	case "Date.DateTime":
		if len(args) > 3 {
			return "", fmt.Errorf("Date.DateTime: expected 0 to 3 arguments, got %d", len(args))
		}
		fromParam := argOr(args, 0, "")
		toParam := argOr(args, 1, "")
		formatParam := argOr(args, 2, "")
		return f.kmock.Date.DateTime(fromParam, toParam, formatParam), nil

	case "Date.Now":
		if len(args) > 1 {
			return "", fmt.Errorf("Date.Now: expected 0 or 1 argument, got %d", len(args))
		}
		formatParam := argOr(args, 0, "")
		return f.kmock.Date.Now(formatParam), nil

	/*
		FILES
	*/
	case "File.Name":
		return f.kmock.File.Name(), nil

	case "File.Extension":
		return f.kmock.File.Extension(), nil

	case "File.NameWithExtension":
		return f.kmock.File.NameWithExtension(), nil

	case "File.MimeType":
		return f.kmock.File.MimeType(), nil

	/*
		FINANCES
	*/
	case "Finance.CreditCardVendor":
		return f.kmock.Finance.CreditCardVendor(), nil

	case "Finance.CreditCardNumber":
		return f.kmock.Finance.CreditCardNumber(), nil

	case "Finance.CreditCardCVV":
		return f.kmock.Finance.CreditCardCVV(), nil

	case "Finance.CreditCardExpirationDate":
		return f.kmock.Finance.CreditCardExpirationDate(), nil

	case "Finance.CreditCardHolder":
		if len(args) > 1 {
			return "", fmt.Errorf("Finance.CreditCardHolder: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		holder, err := f.kmock.Finance.CreditCardHolder(localeParam)
		if err != nil {
			return "", fmt.Errorf("Finance.CreditCardHolder: %v", err)
		}
		return holder, nil

	/*
		IDS
	*/
	case "ID.UUIDv4":
		uuid, err := f.kmock.ID.UUIDv4()
		if err != nil {
			return "", fmt.Errorf("ID.UUIDv4: %v", err)
		}
		return uuid, nil

	case "ID.UUIDv6":
		uuid, err := f.kmock.ID.UUIDv6()
		if err != nil {
			return "", fmt.Errorf("ID.UUIDv6: %v", err)
		}
		return uuid, nil

	case "ID.UUIDv7":
		uuid, err := f.kmock.ID.UUIDv7()
		if err != nil {
			return "", fmt.Errorf("ID.UUIDv7: %v", err)
		}
		return uuid, nil

	case "ID.SequentialID":
		if len(args) > 1 {
			return "", fmt.Errorf("ID.SequentialID: expected 0 or 1 argument, got %d", len(args))
		}
		startFromParam := argOr(args, 0, 0)
		return strconv.Itoa(f.kmock.ID.SequentialID(startFromParam)), nil

	/*
		INTERNET
	*/
	case "Internet.Domain":
		return f.kmock.Internet.Domain(), nil

	case "Internet.DomainWithSubdomain":
		return f.kmock.Internet.DomainWithSubdomain(), nil

	case "Internet.Ipv4":
		return f.kmock.Internet.Ipv4(), nil

	case "Internet.Ipv6":
		return f.kmock.Internet.Ipv6(), nil

	case "Internet.MacAddress":
		return f.kmock.Internet.MacAddress(), nil

	case "Internet.Url":
		return f.kmock.Internet.Url(), nil

	case "Internet.UserAgent":
		return f.kmock.Internet.UserAgent(), nil

	/*
		LOREM
	*/
	case "Lorem.Word":
		return f.kmock.Lorem.Word(), nil

	case "Lorem.Sentence":
		if len(args) > 1 {
			return "", fmt.Errorf("Lorem.Sentence: expected 0 or 1 argument, got %d", len(args))
		}
		wordCountParam := argOr(args, 0, 0)
		return f.kmock.Lorem.Sentence(wordCountParam), nil

	case "Lorem.Paragraph":
		if len(args) > 1 {
			return "", fmt.Errorf("Lorem.Paragraph: expected 0 or 1 argument, got %d", len(args))
		}
		sentenceCountParam := argOr(args, 0, 0)
		return f.kmock.Lorem.Paragraph(sentenceCountParam), nil

	/*
		NUMBERS
	*/
	case "Number.Int":
		return strconv.Itoa(f.kmock.Number.Int()), nil

	case "Number.IntBetween":
		if len(args) > 2 {
			return "", fmt.Errorf("Number.IntBetween: expected 0, 1 or 2 arguments, got %d", len(args))
		}
		minParam := argOr(args, 0, 0)
		maxParam := argOr(args, 1, 0)
		return strconv.Itoa(f.kmock.Number.IntBetween(minParam, maxParam)), nil

	case "Number.Float":
		return strconv.FormatFloat(f.kmock.Number.Float(), 'f', 6, 64), nil

	case "Number.FloatBetween":
		if len(args) > 3 {
			return "", fmt.Errorf("Number.FloatBetween: expected 0 to 3 arguments, got %d", len(args))
		}
		decimalsParam := argOr(args, 0, 2)
		minParam := argOr(args, 1, 0.0)
		maxParam := argOr(args, 2, 0.0)
		return strconv.FormatFloat(f.kmock.Number.FloatBetween(decimalsParam, minParam, maxParam), 'f', decimalsParam, 64), nil

	/*
		PERSON
	*/
	case "Person.Name":
		if len(args) > 1 {
			return "", fmt.Errorf("Person.Name: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		name, err := f.kmock.Person.Name(localeParam)
		if err != nil {
			return "", fmt.Errorf("Person.Name: %v", err)
		}
		return name, nil

	case "Person.FirstName":
		if len(args) > 1 {
			return "", fmt.Errorf("Person.FirstName: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		firstName, err := f.kmock.Person.FirstName(localeParam)
		if err != nil {
			return "", fmt.Errorf("Person.FirstName: %v", err)
		}
		return firstName, nil

	case "Person.MiddleName":
		if len(args) > 1 {
			return "", fmt.Errorf("Person.MiddleName: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		middleName, err := f.kmock.Person.MiddleName(localeParam)
		if err != nil {
			return "", fmt.Errorf("Person.MiddleName: %v", err)
		}
		return middleName, nil

	case "Person.LastName":
		if len(args) > 1 {
			return "", fmt.Errorf("Person.LastName: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		lastName, err := f.kmock.Person.LastName(localeParam)
		if err != nil {
			return "", fmt.Errorf("Person.LastName: %v", err)
		}
		return lastName, nil

	case "Person.Phone":
		return f.kmock.Person.Phone(), nil

	case "Person.Email":
		email, err := f.kmock.Person.Email()
		if err != nil {
			return "", fmt.Errorf("Person.Email: %v", err)
		}
		return email, nil

	case "Person.Username":
		username, err := f.kmock.Person.Username()
		if err != nil {
			return "", fmt.Errorf("Person.Username: %v", err)
		}
		return username, nil

	case "Person.Password":
		if len(args) > 1 {
			return "", fmt.Errorf("Person.Password: expected 0 or 1 argument, got %d", len(args))
		}
		strengthParam := argOr(args, 0, "")
		password, err := f.kmock.Person.Password(strengthParam)
		if err != nil {
			return "", fmt.Errorf("Person.Password: %v", err)
		}
		return password, nil

	case "Person.JobTitle":
		if len(args) > 1 {
			return "", fmt.Errorf("Person.JobTitle: expected 0 or 1 argument, got %d", len(args))
		}
		localeParam := argOr(args, 0, kmock.EN_US)
		jobTitle, err := f.kmock.Person.JobTitle(localeParam)
		if err != nil {
			return "", fmt.Errorf("Person.JobTitle: %v", err)
		}
		return jobTitle, nil

	case "Person.CPFValid":
		return f.kmock.Person.CPFValid(), nil

	case "Person.CPFInvalid":
		return f.kmock.Person.CPFInvalid(), nil

	/*
		REGEX
	*/
	case "Regex.Generate":
		if len(args) != 1 {
			return "", fmt.Errorf("Regex.Generate: expected 1 argument, got %d", len(args))
		}
		pattern, err := parseRegexArg(args[0])
		if err != nil {
			return "", fmt.Errorf("Regex.Generate: %v", err)
		}
		result, err := f.kmock.Regex.Generate(pattern)
		if err != nil {
			return "", fmt.Errorf("Regex.Generate: %v", err)
		}
		return result, nil

	default:
		return "", fmt.Errorf("unknown mock function '%s'", fn)
	}
}

// DocsToTable converts the runtime documentation of available mock functions into a table format (slice of string slices) where each inner slice contains the function signature and its description.
func (f Faker) DocsToTable() [][]string {
	docs := f.kmock.RuntimeDocs()
	table := make([][]string, len(docs))
	for i, doc := range docs {
		args := ""
		for j, arg := range doc.Params {
			if j > 0 {
				args += ":"
			}
			args += fmt.Sprintf("{%s}", arg)
		}
		table[i] = []string{fmt.Sprintf("%s.%s%s", doc.Domain, doc.Method, args), doc.Description}
	}
	return table
}

// argOr is a helper function that retrieves an argument from the args slice at the specified index and converts it to the desired type T (string, int, or float64).
// If the argument is not present or cannot be converted, it returns the provided default value.
func argOr[T string | int | float64](args []string, index int, defaultVal T) T {
	switch any(defaultVal).(type) {
	case string:
		if index < len(args) && args[index] != "" {
			return any(args[index]).(T)
		}
	case int:
		if index < len(args) && args[index] != "" {
			parsed, err := strconv.Atoi(args[index])
			if err == nil {
				return any(parsed).(T)
			}
		}
	case float64:
		if index < len(args) && args[index] != "" {
			parsed, err := strconv.ParseFloat(args[index], 64)
			if err == nil {
				return any(parsed).(T)
			}
		}
	}
	return defaultVal
}

// parseRegexArg is a helper function that validates and extracts the regex pattern from the provided argument string.
// The argument must be wrapped with '/' characters (e.g., "/^[a-z]+$/").
// If the argument is valid, it returns the extracted regex pattern; otherwise, it returns an error.
func parseRegexArg(arg string) (string, error) {
	if len(arg) < 2 {
		return "", fmt.Errorf("invalid regex pattern '%s': must be wrapped with '/'", arg)
	}
	if arg[0] != '/' || arg[len(arg)-1] != '/' {
		return "", fmt.Errorf("invalid regex pattern '%s': must start and end with '/'", arg)
	}
	return arg[1 : len(arg)-1], nil
}
