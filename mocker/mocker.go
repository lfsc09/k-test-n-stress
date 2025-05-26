package mocker

import (
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"

	"github.com/jaswdr/faker/v2"
	regen "github.com/zach-klippenstein/goregen"
)

type Mocker interface {
	List(out io.Writer)
	Generate(mockFunction string, functionParams []string) (string, error)
}

type Mock struct {
	jaswdrFaker *faker.Faker
}

func New() *Mock {
	jaswdrFaker := faker.New()

	return &Mock{
		jaswdrFaker: &jaswdrFaker,
	}
}

func tableLineDivider(colSizes []int) string {
	var line string
	for idx, size := range colSizes {
		if idx == 0 {
			line += strings.Repeat("-", size)
		} else {
			line += "+" + strings.Repeat("-", size)
		}
	}
	return line
}

func tableLineHeader(colSizes []int) string {
	var line string
	for idx, size := range colSizes {
		if idx == 0 {
			line += fmt.Sprintf("%-*s", size, "FUNCTION")
		} else {
			line += "| " + fmt.Sprintf("%-*s", size, "DESCRIPTION")
		}
	}
	return line
}

func tableLineData(colSizes []int, data []string) string {
	var line string
	for idx, size := range colSizes {
		if idx == 0 {
			line += fmt.Sprintf("%-*s", size, data[idx])
		} else {
			line += "| " + fmt.Sprintf("%-*s", size, data[idx])
		}
	}
	return line
}

// fmt.Fprintf(out, "%-40s | %s\n", "Address.latitude", "Generates a random latitude")
func (m *Mock) List(out io.Writer) {
	colSizes := []int{40, 60}
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineHeader(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Address.latitude", "Generates a random latitude"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Address.longitude", "Generates a random longitude"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Address.postCode", "Generates a random post code"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Address.country", "Generates a random country"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Address.state", "Generates a random state"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Address.city", "Generates a random city"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Address.streetName", "Generates a random street name"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Address.buildingNumber", "Generates a random building number"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Boolean.boolean", "Generates a random boolean"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Boolean.booleanWithChance:[chance]", "Generates a random boolean with a chance of true"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Car.maker", "Generates a random car maker"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Car.model", "Generates a random car model"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Car.plate", "Generates a random car plate"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Company.name", "Generates a random company name"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Company.suffix", "Generates a random company suffix"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Company.catchPhrase", "Generates a random company catch phrase"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Company.bs", "Generates a random company BS"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Company.jobTitle", "Generates a random company job title"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Company.cnpj", "Generates a random valid brazilian cnpj"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Currency.currencyCode", "Generates a random currency code"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Currency.currencyContry", "Generates a random currency country"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Currency.currencyName", "Generates a random currency name"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Currency.currencyNumber", "Generates a random currency number"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"File.filenameWithExtension", "Generates a random filename with extension"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"File.extension", "Generates a random file extension"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Internet.domain", "Generates a random domain"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Internet.email", "Generates a random email"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Internet.ipv4", "Generates a random IPv4 address"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Internet.macAddress", "Generates a random MAC address"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Internet.password", "Generates a random password"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Internet.url", "Generates a random URL"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Lorem.paragraph:[sentences]", "Generates a random paragraph with N number of sentences"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Lorem.paragraphs:[paragraphs]", "Generates N number of random paragraphs"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Lorem.sentence:[words]", "Generates a random sentence with N number of words"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Lorem.sentences:[sentences]", "Generates N number of random sentences"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Lorem.word", "Generates a random word"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Lorem.words:[words]", "Generates N number of random words"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Number.number:[decimals]:[min]:[max]", "Generates a random number with N decimals, from min up to max"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Payment.creditCardExpirationDate", "Generates a random credit card expiration date"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Payment.creditCardNumber", "Generates a random credit card number"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Payment.creditCardType", "Generates a random credit card type"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Payment.creditCardCvv", "Generates a random credit card CVV"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Person.phoneNumber", "Generates a random phone number"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Person.email", "Generates a random email"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Person.firstName", "Generates a random first name"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Person.lastName", "Generates a random last name"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Person.name", "Generates a random name"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Person.cpf", "Generates a random valid brazilian cpf"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Regex.regex:[regex]", "Generates a random string based on the regex pattern"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Time.date", "Generates a random date"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"UUID.uuidv4", "Generates a random UUID v4"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"UserAgent.userAgent", "Generates a random user agent"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
}

func (m *Mock) Generate(mockFunction string, functionParams []string) (string, error) {
	switch mockFunction {
	/*
		ADDRESSES
	*/
	case "Address.latitude":
		return strconv.FormatFloat(m.jaswdrFaker.Address().Latitude(), 'f', 6, 64), nil
	case "Address.longitude":
		return strconv.FormatFloat(m.jaswdrFaker.Address().Longitude(), 'f', 6, 64), nil
	case "Address.postCode":
		return m.jaswdrFaker.Address().PostCode(), nil
	case "Address.country":
		return m.jaswdrFaker.Address().Country(), nil
	case "Address.state":
		return m.jaswdrFaker.Address().State(), nil
	case "Address.city":
		return m.jaswdrFaker.Address().City(), nil
	case "Address.streetName":
		return m.jaswdrFaker.Address().StreetName(), nil
	case "Address.buildingNumber":
		return m.jaswdrFaker.Address().BuildingNumber(), nil
	/*
		BOOLEANS
	*/
	case "Boolean.boolean":
		return strconv.FormatBool(m.jaswdrFaker.Boolean().Bool()), nil
	case "Boolean.booleanWithChance":
		chance, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return strconv.FormatBool(m.jaswdrFaker.Boolean().Bool()), nil
		}
		return strconv.FormatBool(m.jaswdrFaker.Boolean().BoolWithChance(chance)), nil
	/*
		CAR
	*/
	case "Car.maker":
		return m.jaswdrFaker.Car().Maker(), nil
	case "Car.model":
		return m.jaswdrFaker.Car().Model(), nil
	case "Car.plate":
		return m.jaswdrFaker.Car().Plate(), nil
	/*
		COMPANY
	*/
	case "Company.name":
		return m.jaswdrFaker.Company().Name(), nil
	case "Company.suffix":
		return m.jaswdrFaker.Company().Suffix(), nil
	case "Company.catchPhrase":
		return m.jaswdrFaker.Company().CatchPhrase(), nil
	case "Company.bs":
		return m.jaswdrFaker.Company().BS(), nil
	case "Company.jobTitle":
		return m.jaswdrFaker.Company().JobTitle(), nil
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
		return m.jaswdrFaker.Currency().Code(), nil
	case "Currency.currencyContry":
		return m.jaswdrFaker.Currency().Country(), nil
	case "Currency.currencyName":
		return m.jaswdrFaker.Currency().Currency(), nil
	case "Currency.currencyNumber":
		return strconv.Itoa(m.jaswdrFaker.Currency().Number()), nil
	/*
		FILE
	*/
	case "File.filenameWithExtension":
		return m.jaswdrFaker.File().FilenameWithExtension(), nil
	case "File.extension":
		return m.jaswdrFaker.File().Extension(), nil
	/*
		INTERNET
	*/
	case "Internet.domain":
		return m.jaswdrFaker.Internet().Domain(), nil
	case "Internet.email":
		return m.jaswdrFaker.Internet().Email(), nil
	case "Internet.ipv4":
		return m.jaswdrFaker.Internet().Ipv4(), nil
	case "Internet.macAddress":
		return m.jaswdrFaker.Internet().MacAddress(), nil
	case "Internet.password":
		return m.jaswdrFaker.Internet().Password(), nil
	case "Internet.url":
		return m.jaswdrFaker.Internet().URL(), nil
	/*
		LOREM
	*/
	case "Lorem.paragraph":
		sentences, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return m.jaswdrFaker.Lorem().Paragraph(1), nil
		}
		return m.jaswdrFaker.Lorem().Paragraph(sentences), nil
	case "Lorem.paragraphs":
		paragraphs, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return strings.Join(m.jaswdrFaker.Lorem().Paragraphs(1), ""), nil
		}
		return strings.Join(m.jaswdrFaker.Lorem().Paragraphs(paragraphs), "\n"), nil
	case "Lorem.sentence":
		words, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return m.jaswdrFaker.Lorem().Sentence(1), nil
		}
		return m.jaswdrFaker.Lorem().Sentence(words), nil
	case "Lorem.sentences":
		sentences, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return strings.Join(m.jaswdrFaker.Lorem().Sentences(1), ""), nil
		}
		return strings.Join(m.jaswdrFaker.Lorem().Sentences(sentences), "\n"), nil
	case "Lorem.word":
		return m.jaswdrFaker.Lorem().Word(), nil
	case "Lorem.words":
		words, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return strings.Join(m.jaswdrFaker.Lorem().Words(1), " "), nil
		}
		return strings.Join(m.jaswdrFaker.Lorem().Words(words), ""), nil
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
		return strconv.FormatFloat(m.jaswdrFaker.Float64(decimals, int(min), int(max)), 'f', decimals, 64), nil
	/*
		PAYMENT
	*/
	case "Payment.creditCardExpirationDate":
		return m.jaswdrFaker.Payment().CreditCardExpirationDateString(), nil
	case "Payment.creditCardNumber":
		return m.jaswdrFaker.Payment().CreditCardNumber(), nil
	case "Payment.creditCardType":
		return m.jaswdrFaker.Payment().CreditCardType(), nil
	case "Payment.creditCardCvv":
		cvv, err := regen.Generate("[0-9]{3}")
		if err != nil {
			return "", fmt.Errorf("failed to generate CVV '%w'", err)
		}
		return cvv, nil
	/*
		PERSON
	*/
	case "Person.phoneNumber":
		return m.jaswdrFaker.Person().Contact().Phone, nil
	case "Person.email":
		return m.jaswdrFaker.Person().Contact().Email, nil
	case "Person.firstName":
		return m.jaswdrFaker.Person().FirstName(), nil
	case "Person.lastName":
		return m.jaswdrFaker.Person().LastName(), nil
	case "Person.name":
		return m.jaswdrFaker.Person().Name(), nil
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
			return "", fmt.Errorf("regex function requires a regex pattern as parameter")
		}
		regex, err := extractRegex(functionParams[0])
		if err != nil {
			return "", err
		}
		randomRegex, err := regen.Generate(regex)
		if err != nil {
			return "", fmt.Errorf("failed to generate regex '%w'", err)
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
		return m.jaswdrFaker.UUID().V4(), nil
	/*
		USER AGENT
	*/
	case "UserAgent.userAgent":
		return m.jaswdrFaker.UserAgent().UserAgent(), nil
	default:
		return "", fmt.Errorf("unknown mock function '%s'", mockFunction)
	}
}
