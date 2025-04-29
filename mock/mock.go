package mock

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/jaswdr/faker/v2"
	regen "github.com/zach-klippenstein/goregen"
)

type Mocker interface {
	List()
	Generate(mockKey string, mockValues interface{}) interface{}
}

type Mock struct {
	faker *faker.Faker
}

func New() *Mock {
	faker := faker.New()

	return &Mock{
		faker: &faker,
	}
}

func (m *Mock) List() {
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("ADDRESS functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "Address.latitude", "Generates a random latitude")
	fmt.Printf("%-40s | %s\n", "Address.longitude", "Generates a random longitude")
	fmt.Printf("%-40s | %s\n", "Address.postCode", "Generates a random post code")
	fmt.Printf("%-40s | %s\n", "Address.country", "Generates a random country")
	fmt.Printf("%-40s | %s\n", "Address.state", "Generates a random state")
	fmt.Printf("%-40s | %s\n", "Address.city", "Generates a random city")
	fmt.Printf("%-40s | %s\n", "Address.streetName", "Generates a random street name")
	fmt.Printf("%-40s | %s\n", "Address.buildingNumber", "Generates a random building number")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("BOOLEAN functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "Boolean.boolean", "Generates a random boolean")
	fmt.Printf("%-40s | %s\n", "Boolean.booleanWithChance:[chance]", "Generates a random boolean with a chance of true")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("CAR functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "Car.maker", "Generates a random car maker")
	fmt.Printf("%-40s | %s\n", "Car.model", "Generates a random car model")
	fmt.Printf("%-40s | %s\n", "Car.plate", "Generates a random car plate")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("COMPANY functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "Company.name", "Generates a random company name")
	fmt.Printf("%-40s | %s\n", "Company.suffix", "Generates a random company suffix")
	fmt.Printf("%-40s | %s\n", "Company.catchPhrase", "Generates a random company catch phrase")
	fmt.Printf("%-40s | %s\n", "Company.bs", "Generates a random company BS")
	fmt.Printf("%-40s | %s\n", "Company.jobTitle", "Generates a random company job title")
	fmt.Printf("%-40s | %s\n", "Company.cnpj", "Generates a random valid CNPJ")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("CURRENCY functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "Currency.currencyCode", "Generates a random currency code")
	fmt.Printf("%-40s | %s\n", "Currency.currencyContry", "Generates a random currency country")
	fmt.Printf("%-40s | %s\n", "Currency.currencyName", "Generates a random currency name")
	fmt.Printf("%-40s | %s\n", "Currency.currencyNumber", "Generates a random currency number")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("FILE functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "File.filenameWithExtension", "Generates a random filename with extension")
	fmt.Printf("%-40s | %s\n", "File.extension", "Generates a random file extension")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("INTERNET functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "Internet.domain", "Generates a random domain")
	fmt.Printf("%-40s | %s\n", "Internet.email", "Generates a random email")
	fmt.Printf("%-40s | %s\n", "Internet.ipv4", "Generates a random IPv4 address")
	fmt.Printf("%-40s | %s\n", "Internet.macAddress", "Generates a random MAC address")
	fmt.Printf("%-40s | %s\n", "Internet.password", "Generates a random password")
	fmt.Printf("%-40s | %s\n", "Internet.url", "Generates a random URL")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("LOREM functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "Lorem.paragraph:[sentences]", "Generates a random paragraph with N number of sentences")
	fmt.Printf("%-40s | %s\n", "Lorem.paragraphs:[paragraphs]", "Generates N number of random paragraphs")
	fmt.Printf("%-40s | %s\n", "Lorem.sentence:[words]", "Generates a random sentence with N number of words")
	fmt.Printf("%-40s | %s\n", "Lorem.sentences:[sentences]", "Generates N number of random sentences")
	fmt.Printf("%-40s | %s\n", "Lorem.word", "Generates a random word")
	fmt.Printf("%-40s | %s\n", "Lorem.words:[words]", "Generates N number of random words")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("NUMBER functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "Number.number:[decimals]:[min]:[max]", "Generates a random number with N decimals, from min up to max")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("PAYMENT functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "Payment.creditCardExpirationDate", "Generates a random credit card expiration date")
	fmt.Printf("%-40s | %s\n", "Payment.creditCardNumber", "Generates a random credit card number")
	fmt.Printf("%-40s | %s\n", "Payment.creditCardType", "Generates a random credit card type")
	fmt.Printf("%-40s | %s\n", "Payment.creditCardCvv", "Generates a random credit card CVV")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("PERSON functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "Person.phoneNumber", "Generates a random phone number")
	fmt.Printf("%-40s | %s\n", "Person.email", "Generates a random email")
	fmt.Printf("%-40s | %s\n", "Person.firstName", "Generates a random first name")
	fmt.Printf("%-40s | %s\n", "Person.lastName", "Generates a random last name")
	fmt.Printf("%-40s | %s\n", "Person.name", "Generates a random name")
	fmt.Printf("%-40s | %s\n", "Person.cpf", "Generates a random valid CPF")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("REGEX functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "Regex.regex:[regex]", "Generates a random string based on the regex pattern")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("TIME functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "Time.date", "Generates a random date")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("UUID functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "UUID.uuidv4", "Generates a random UUID v4")
	fmt.Println("\n" + strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("USER AGENT functions:\n")
	fmt.Println(strings.Repeat("-", 40) + "-+-" + strings.Repeat("-", 60))
	fmt.Printf("%-40s | %s\n", "UserAgent.userAgent", "Generates a random user agent")
	fmt.Printf("\n")
}

func (m *Mock) Generate(mockFunction string, functionParams []string) (string, error) {
	switch mockFunction {
	/*
		ADDRESSES
	*/
	case "Address.latitude":
		return strconv.FormatFloat(m.faker.Address().Latitude(), 'f', 6, 64), nil
	case "Address.longitude":
		return strconv.FormatFloat(m.faker.Address().Longitude(), 'f', 6, 64), nil
	case "Address.postCode":
		return m.faker.Address().PostCode(), nil
	case "Address.country":
		return m.faker.Address().Country(), nil
	case "Address.state":
		return m.faker.Address().State(), nil
	case "Address.city":
		return m.faker.Address().City(), nil
	case "Address.streetName":
		return m.faker.Address().StreetName(), nil
	case "Address.buildingNumber":
		return m.faker.Address().BuildingNumber(), nil
	/*
		BOOLEANS
	*/
	case "Boolean.boolean":
		return strconv.FormatBool(m.faker.Boolean().Bool()), nil
	case "Boolean.booleanWithChance":
		chance, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return strconv.FormatBool(m.faker.Boolean().Bool()), nil
		}
		return strconv.FormatBool(m.faker.Boolean().BoolWithChance(chance)), nil
	/*
		CAR
	*/
	case "Car.maker":
		return m.faker.Car().Maker(), nil
	case "Car.model":
		return m.faker.Car().Model(), nil
	case "Car.plate":
		return m.faker.Car().Plate(), nil
	/*
		COMPANY
	*/
	case "Company.name":
		return m.faker.Company().Name(), nil
	case "Company.suffix":
		return m.faker.Company().Suffix(), nil
	case "Company.catchPhrase":
		return m.faker.Company().CatchPhrase(), nil
	case "Company.bs":
		return m.faker.Company().BS(), nil
	case "Company.jobTitle":
		return m.faker.Company().JobTitle(), nil
	case "Company.cnpj":
		cnpj := make([]int, 12)

		// Generate the first 12 random digits
		for i := range 12 {
			cnpj[i] = rand.Intn(10)
		}

		// Multipliers for checksum digits
		multipliers1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
		multipliers2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

		// Calculate checksums and append them
		cnpj = append(cnpj, calculateChecksum(cnpj, multipliers1))
		cnpj = append(cnpj, calculateChecksum(cnpj, multipliers2))

		// Format the CNPJ
		return fmt.Sprintf("%02d.%03d.%03d/%04d-%02d",
			cnpj[0]*10+cnpj[1],
			cnpj[2]*100+cnpj[3]*10+cnpj[4],
			cnpj[5]*100+cnpj[6]*10+cnpj[7],
			cnpj[8]*1000+cnpj[9]*100+cnpj[10]*10+cnpj[11],
			cnpj[12]*10+cnpj[13],
		), nil
	/*
		CURRENCY
	*/
	case "Currency.currencyCode":
		return m.faker.Currency().Code(), nil
	case "Currency.currencyContry":
		return m.faker.Currency().Country(), nil
	case "Currency.currencyName":
		return m.faker.Currency().Currency(), nil
	case "Currency.currencyNumber":
		return strconv.Itoa(m.faker.Currency().Number()), nil
	/*
		FILE
	*/
	case "File.filenameWithExtension":
		return m.faker.File().FilenameWithExtension(), nil
	case "File.extension":
		return m.faker.File().Extension(), nil
	/*
		INTERNET
	*/
	case "Internet.domain":
		return m.faker.Internet().Domain(), nil
	case "Internet.email":
		return m.faker.Internet().Email(), nil
	case "Internet.ipv4":
		return m.faker.Internet().Ipv4(), nil
	case "Internet.macAddress":
		return m.faker.Internet().MacAddress(), nil
	case "Internet.password":
		return m.faker.Internet().Password(), nil
	case "Internet.url":
		return m.faker.Internet().URL(), nil
	/*
		LOREM
	*/
	case "Lorem.paragraph":
		sentences, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return m.faker.Lorem().Paragraph(1), nil
		}
		return m.faker.Lorem().Paragraph(sentences), nil
	case "Lorem.paragraphs":
		paragraphs, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return strings.Join(m.faker.Lorem().Paragraphs(1), ""), nil
		}
		return strings.Join(m.faker.Lorem().Paragraphs(paragraphs), "\n"), nil
	case "Lorem.sentence":
		words, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return m.faker.Lorem().Sentence(1), nil
		}
		return m.faker.Lorem().Sentence(words), nil
	case "Lorem.sentences":
		sentences, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return strings.Join(m.faker.Lorem().Sentences(1), ""), nil
		}
		return strings.Join(m.faker.Lorem().Sentences(sentences), "\n"), nil
	case "Lorem.word":
		return m.faker.Lorem().Word(), nil
	case "Lorem.words":
		words, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return strings.Join(m.faker.Lorem().Words(1), " "), nil
		}
		return strings.Join(m.faker.Lorem().Words(words), ""), nil
	/*
		NUMBER
	*/
	case "Number.number":
		decimals := 0
		min := -1000.0
		max := 1000.0
		if len(functionParams) > 0 && functionParams[0] != "" {
			decimals, _ = strconv.Atoi(functionParams[0])
		}
		if len(functionParams) > 1 && functionParams[1] != "" {
			min, _ = strconv.ParseFloat(functionParams[1], 64)
		}
		if len(functionParams) > 2 && functionParams[2] != "" {
			max, _ = strconv.ParseFloat(functionParams[2], 64)
		}
		return strconv.FormatFloat(m.faker.Float64(decimals, int(min), int(max)), 'f', decimals, 64), nil
	/*
		PAYMENT
	*/
	case "Payment.creditCardExpirationDate":
		return m.faker.Payment().CreditCardExpirationDateString(), nil
	case "Payment.creditCardNumber":
		return m.faker.Payment().CreditCardNumber(), nil
	case "Payment.creditCardType":
		return m.faker.Payment().CreditCardType(), nil
	case "Payment.creditCardCvv":
		cvv, err := regen.Generate("[0-9]{3}")
		if err != nil {
			return "", fmt.Errorf("Failed to generate CVV: %v\n", err)
		}
		return cvv, nil
	/*
		PERSON
	*/
	case "Person.phoneNumber":
		return m.faker.Person().Contact().Phone, nil
	case "Person.email":
		return m.faker.Person().Contact().Email, nil
	case "Person.firstName":
		return m.faker.Person().FirstName(), nil
	case "Person.lastName":
		return m.faker.Person().LastName(), nil
	case "Person.name":
		return m.faker.Person().Name(), nil
	case "Person.cpf":
		cpf := make([]int, 9)

		// Generate the first 9 random digits
		for i := range 9 {
			cpf[i] = rand.Intn(10)
		}

		// Multipliers for checksum digits
		multipliers1 := []int{10, 9, 8, 7, 6, 5, 4, 3, 2}
		multipliers2 := []int{11, 10, 9, 8, 7, 6, 5, 4, 3, 2}

		// Calculate checksums and append them
		cpf = append(cpf, calculateChecksum(cpf, multipliers1))
		cpf = append(cpf, calculateChecksum(cpf, multipliers2))

		// Format the CPF
		return fmt.Sprintf("%03d.%03d.%03d-%02d",
			cpf[0]*100+cpf[1]*10+cpf[2],
			cpf[3]*100+cpf[4]*10+cpf[5],
			cpf[6]*100+cpf[7]*10+cpf[8],
			cpf[9]*10+cpf[10],
		), nil
	/*
		REGEX
	*/
	case "Regex.regex":
		if len(functionParams) == 0 {
			return "", fmt.Errorf("Regex function requires a regex pattern as parameter")
		}
		regex, err := extractRegex(functionParams[0])
		if err != nil {
			return "", err
		}
		randomRegex, err := regen.Generate(regex)
		if err != nil {
			return "", fmt.Errorf("Failed to generate regex: %v\n", err)
		}
		return randomRegex, nil
	/*
		TIME
	*/
	case "Time.date":
		return "", nil
	/*
		UUID
	*/
	case "UUID.uuidv4":
		return m.faker.UUID().V4(), nil
	/*
		USER AGENT
	*/
	case "UserAgent.userAgent":
		return m.faker.UserAgent().UserAgent(), nil
	default:
		return "", fmt.Errorf("Unknown mock function: %v\n", mockFunction)
	}
}

// Helper function to calculate checksum for CPF and CNPJ
func calculateChecksum(digits []int, multipliers []int) int {
	sum := 0
	for i := range digits {
		sum += digits[i] * multipliers[i]
	}
	remainder := sum % 11
	if remainder < 2 {
		return 0
	}
	return 11 - remainder
}

// Extracts raw regex string from /.../ and unescapes \/ → /
func extractRegex(value string) (string, error) {
	if !strings.HasPrefix(value, "/") || !strings.HasSuffix(value, "/") {
		return "", fmt.Errorf("Value '%s' must be wrapped in /.../", value)
	}
	trimmed := value[1 : len(value)-1]
	unescaped := strings.ReplaceAll(trimmed, `\/`, `/`)
	return unescaped, nil
}
