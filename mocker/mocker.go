package mocker

import (
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jaswdr/faker/v2"
)

// mockerSeedCounter provides monotonically-increasing offsets so concurrent
// New() calls never collide on the same UnixNano value.
var mockerSeedCounter atomic.Int64

type Mock struct {
	jaswdrFaker *faker.Faker
	rng         *rand.Rand        // per-instance source; never share across goroutines
	cvvGen      *RandexpGenerator // pre-built generator for credit card CVV, since it's a common use case and has a simple fixed pattern
}

// New creates a new instance of Mock with an initialized jaswdrFaker and a
// per-instance RNG seeded with a unique value derived from the current time,
// the PID, and an atomic counter so concurrent calls cannot share a seed.
func New() *Mock {
	jaswdrFaker := faker.New()
	// Fibonacci hashing multiplier (signed): ensures unique seeds across concurrent New() calls
	seed := time.Now().UnixNano() ^ (int64(os.Getpid()) * 0x517cc1b727220a95) ^ (mockerSeedCounter.Add(1) * -7046029254386353131)
	rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed)))

	cvvGen, cvvError := NewRandexpGenerator("[0-9]{3}")
	if cvvError != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to build CVV generator: %v\n", cvvError)
	}

	return &Mock{
		jaswdrFaker: &jaswdrFaker,
		rng:         rng,
		cvvGen:      cvvGen,
	}
}

// tableLineDivider generates a string that represents a divider line for a table based on the provided column sizes.
func tableLineDivider(colSizes []int) string {
	var line strings.Builder
	for idx, size := range colSizes {
		if idx == 0 {
			line.WriteString(strings.Repeat("-", size))
		} else {
			line.WriteString("+" + strings.Repeat("-", size))
		}
	}
	return line.String()
}

// tableLineHeader generates a string that represents the header line for a table based on the provided column sizes.
func tableLineHeader(colSizes []int) string {
	var line strings.Builder
	for idx, size := range colSizes {
		if idx == 0 {
			fmt.Fprintf(&line, "%-*s", size, "FUNCTION")
		} else {
			fmt.Fprintf(&line, "| %-*s", size, "DESCRIPTION")
		}
	}
	return line.String()
}

// tableLineData generates a string that represents a data line for a table based on the provided column sizes and data.
func tableLineData(colSizes []int, data []string) string {
	var line strings.Builder
	for idx, size := range colSizes {
		if idx == 0 {
			fmt.Fprintf(&line, "%-*s", size, data[idx])
		} else {
			fmt.Fprintf(&line, "| %-*s", size, data[idx])
		}
	}
	return line.String()
}

// List writes a formatted list of available mock functions and their descriptions to the provided io.Writer. Each function is displayed in a tabular format with its name and description.
// e.g.: fmt.Fprintf(out, "%-40s | %s\n", "Address.latitude", "Generates a random latitude")
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
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Boolean.booleanWithChance:{chance}", "Generates a random boolean with a chance of true"}))
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
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Lorem.paragraph:{sentences}", "Generates a random paragraph with N number of sentences"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Lorem.paragraphs:{paragraphs}", "Generates N number of random paragraphs"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Lorem.sentence:{words}", "Generates a random sentence with N number of words"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Lorem.sentences:{sentences}", "Generates N number of random sentences"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Lorem.word", "Generates a random word"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Lorem.words:{words}", "Generates N number of random words"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Number.number:{decimals}:{min}:{max}", "Generates a random number with N decimals, from min up to max"}))
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
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Regex.regex:/regex/", "Generates a random string based on the regex pattern"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Date.date:{from}:{to}:/format/", "Random date. from/to: YYYY-MM-DD. Default format: YYYY-MM-DD"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Date.time:{from}:{to}:/format/", "Random time. from/to: hh:mm. Default format: hh:mm:ss.sss"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Date.datetime:{from}:{to}:/format/", "Random datetime. from/to: YYYY-MM-DDThh:mm. Default: YYYY-MM-DDThh:mm:ss.sss"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"Date.now:/format/", "Current datetime. Default format: YYYY-MM-DDThh:mm:ss.sss"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"UUID.uuidv4", "Generates a random UUID v4"}))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"UUID.uuidv7", "Generates a random UUID v7"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
	fmt.Fprintf(out, "%s\n", tableLineData(colSizes, []string{"UserAgent.userAgent", "Generates a random user agent"}))
	fmt.Fprintf(out, "%s\n", tableLineDivider(colSizes))
}

// Generate takes a mock function name and its parameters, and returns a generated string based on the specified mock function. It uses the jaswdrFaker library to generate various types of random data, such as addresses, booleans, car information, company details, currency information, file names, internet-related data, lorem ipsum text, numbers, payment details, person information, regex-based strings, time data, UUID v4 and UUID v7 values, and user agents. If the provided mock function is not recognized or if there are issues with the parameters, it returns an error.
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
		if len(functionParams) == 0 || functionParams[0] == "" {
			return "", fmt.Errorf("Boolean.booleanWithChance: 'chance' parameter is required")
		}
		chance, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return "", fmt.Errorf("Boolean.booleanWithChance: 'chance' must be an integer, got %q", functionParams[0])
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
			cnpj[i] = m.rng.IntN(10)
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
		if len(functionParams) == 0 || functionParams[0] == "" {
			return "", fmt.Errorf("Lorem.paragraph: 'sentences' parameter is required")
		}
		sentences, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return "", fmt.Errorf("Lorem.paragraph: 'sentences' must be an integer, got %q", functionParams[0])
		}
		return m.jaswdrFaker.Lorem().Paragraph(sentences), nil
	case "Lorem.paragraphs":
		if len(functionParams) == 0 || functionParams[0] == "" {
			return "", fmt.Errorf("Lorem.paragraphs: 'paragraphs' parameter is required")
		}
		paragraphs, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return "", fmt.Errorf("Lorem.paragraphs: 'paragraphs' must be an integer, got %q", functionParams[0])
		}
		return strings.Join(m.jaswdrFaker.Lorem().Paragraphs(paragraphs), "\n"), nil
	case "Lorem.sentence":
		if len(functionParams) == 0 || functionParams[0] == "" {
			return "", fmt.Errorf("Lorem.sentence: 'words' parameter is required")
		}
		words, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return "", fmt.Errorf("Lorem.sentence: 'words' must be an integer, got %q", functionParams[0])
		}
		return m.jaswdrFaker.Lorem().Sentence(words), nil
	case "Lorem.sentences":
		if len(functionParams) == 0 || functionParams[0] == "" {
			return "", fmt.Errorf("Lorem.sentences: 'sentences' parameter is required")
		}
		sentences, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return "", fmt.Errorf("Lorem.sentences: 'sentences' must be an integer, got %q", functionParams[0])
		}
		return strings.Join(m.jaswdrFaker.Lorem().Sentences(sentences), "\n"), nil
	case "Lorem.word":
		return m.jaswdrFaker.Lorem().Word(), nil
	case "Lorem.words":
		if len(functionParams) == 0 || functionParams[0] == "" {
			return "", fmt.Errorf("Lorem.words: 'words' parameter is required")
		}
		words, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return "", fmt.Errorf("Lorem.words: 'words' must be an integer, got %q", functionParams[0])
		}
		return strings.Join(m.jaswdrFaker.Lorem().Words(words), " "), nil
	/*
		NUMBER
	*/
	case "Number.number":
		decimals := 0
		min := -1000.0
		max := 1000.0
		if len(functionParams) > 0 && functionParams[0] != "" {
			v, err := strconv.Atoi(functionParams[0])
			if err != nil {
				return "", fmt.Errorf("Number.number: 'decimals' must be an integer, got %q", functionParams[0])
			}
			decimals = v
		}
		if len(functionParams) > 1 && functionParams[1] != "" {
			v, err := strconv.ParseFloat(functionParams[1], 64)
			if err != nil {
				return "", fmt.Errorf("Number.number: 'min' must be a number, got %q", functionParams[1])
			}
			min = v
		}
		if len(functionParams) > 2 && functionParams[2] != "" {
			v, err := strconv.ParseFloat(functionParams[2], 64)
			if err != nil {
				return "", fmt.Errorf("Number.number: 'max' must be a number, got %q", functionParams[2])
			}
			max = v
		}
		if max < min {
			return "", fmt.Errorf("Number.number: 'max' (%v) must be >= 'min' (%v)", max, min)
		}
		var value float64
		if max == min {
			value = min
		} else {
			value = min + m.rng.Float64()*(max-min)
		}
		return strconv.FormatFloat(value, 'f', decimals, 64), nil
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
		if m.cvvGen == nil {
			return "", fmt.Errorf("missing CVV generator instance")
		}
		return m.cvvGen.Generate(m.rng), nil
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
			cpf[i] = m.rng.IntN(10)
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
		gen, err := NewRandexpGenerator(regex)
		if err != nil {
			return "", fmt.Errorf("failed to generate regex '%w'", err)
		}
		return gen.Generate(m.rng), nil
	/*
		DATE
	*/
	case "Date.date":
		fromDefault := time.Now().AddDate(-5, 0, 0)
		toDefault := time.Now().AddDate(5, 0, 0)
		format := "YYYY-MM-DD"
		fromTime := fromDefault
		toTime := toDefault
		if len(functionParams) > 0 && functionParams[0] != "" {
			if t, err := parseDateOnly(functionParams[0]); err == nil {
				fromTime = t
			}
		}
		if len(functionParams) > 1 && functionParams[1] != "" {
			if t, err := parseDateOnly(functionParams[1]); err == nil {
				toTime = t
			}
		}
		if len(functionParams) > 2 && functionParams[2] != "" {
			extracted, err := extractRegex(functionParams[2])
			if err != nil {
				return "", err
			}
			format = extracted
		}
		delta := toTime.Unix() - fromTime.Unix()
		if delta < 0 {
			return "", fmt.Errorf("Date.date: 'from' must be before 'to'")
		}
		// Pick a random whole day within the range
		daySeconds := int64(24 * 60 * 60)
		days := delta / daySeconds
		randomDay := m.rng.Int64N(days + 1)
		result := fromTime.Add(time.Duration(randomDay*daySeconds) * time.Second)
		return formatDatetime(result, format), nil
	case "Date.time":
		fromDefault, _ := parseTimeOnly("00:00")
		toDefault, _ := parseTimeOnly("23:59")
		format := "hh:mm:ss.sss"
		fromTime := fromDefault
		toTime := toDefault
		if len(functionParams) > 0 && functionParams[0] != "" {
			if t, err := parseTimeOnly(functionParams[0]); err == nil {
				fromTime = t
			}
		}
		if len(functionParams) > 1 && functionParams[1] != "" {
			if t, err := parseTimeOnly(functionParams[1]); err == nil {
				toTime = t
			}
		}
		if len(functionParams) > 2 && functionParams[2] != "" {
			extracted, err := extractRegex(functionParams[2])
			if err != nil {
				return "", err
			}
			format = extracted
		}
		fromSecs := int64(fromTime.Hour()*3600 + fromTime.Minute()*60)
		toSecs := int64(toTime.Hour()*3600 + toTime.Minute()*60)
		totalRange := toSecs - fromSecs + 59 // +59 to include seconds within the final minute
		if totalRange < 0 {
			return "", fmt.Errorf("Date.time: 'from' must be before 'to'")
		}
		randomSecs := m.rng.Int64N(totalRange + 1)
		randomMs := m.rng.IntN(1000)
		h := int((fromSecs + randomSecs) / 3600)
		remaining := (fromSecs + randomSecs) % 3600
		min := int(remaining / 60)
		sec := int(remaining % 60)
		result := time.Date(0, 1, 1, h, min, sec, randomMs*1_000_000, time.UTC)
		return formatDatetime(result, format), nil
	case "Date.datetime":
		fromDefault := time.Now().AddDate(-5, 0, 0)
		toDefault := time.Now().AddDate(5, 0, 0)
		format := "YYYY-MM-DDThh:mm:ss.sss"
		fromTime := fromDefault
		toTime := toDefault
		if len(functionParams) > 0 && functionParams[0] != "" {
			if t, err := parseDatetimeFull(functionParams[0]); err == nil {
				fromTime = t
			}
		}
		if len(functionParams) > 1 && functionParams[1] != "" {
			if t, err := parseDatetimeFull(functionParams[1]); err == nil {
				toTime = t
			}
		}
		if len(functionParams) > 2 && functionParams[2] != "" {
			extracted, err := extractRegex(functionParams[2])
			if err != nil {
				return "", err
			}
			format = extracted
		}
		deltaMs := toTime.UnixMilli() - fromTime.UnixMilli()
		if deltaMs < 0 {
			return "", fmt.Errorf("Date.datetime: 'from' must be before 'to'")
		}
		randomDelta := m.rng.Int64N(deltaMs + 1)
		result := time.UnixMilli(fromTime.UnixMilli() + randomDelta)
		return formatDatetime(result, format), nil
	case "Date.now":
		format := "YYYY-MM-DDThh:mm:ss.sss"
		if len(functionParams) > 0 && functionParams[0] != "" {
			extracted, err := extractRegex(functionParams[0])
			if err != nil {
				return "", err
			}
			format = extracted
		}
		return formatDatetime(time.Now(), format), nil
	/*
		UUID
	*/
	case "UUID.uuidv4":
		return m.jaswdrFaker.UUID().V4(), nil
	case "UUID.uuidv7":
		v7, err := uuid.NewV7()
		if err != nil {
			return "", fmt.Errorf("failed to generate UUID v7: %w", err)
		}
		return v7.String(), nil
	/*
		USER AGENT
	*/
	case "UserAgent.userAgent":
		return m.jaswdrFaker.UserAgent().UserAgent(), nil
	default:
		return "", fmt.Errorf("unknown mock function '%s'", mockFunction)
	}
}
