package mock

import (
	"log"
	"strconv"
	"strings"

	"github.com/jaswdr/faker/v2"
	regen "github.com/zach-klippenstein/goregen"
)

type Mocker interface {
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

func (m *Mock) Generate(mockFunction string, functionParams []string) string {
	switch mockFunction {
	/*
		ADDRESSES
	*/
	case "Address.address":
		return m.faker.Address().Address()
	case "Address.city":
		return m.faker.Address().City()
	case "Address.country":
		return m.faker.Address().Country()
	case "Address.postCode":
		return m.faker.Address().PostCode()
	case "Address.state":
		return m.faker.Address().State()
	case "Address.streetAddress":
		return m.faker.Address().StreetAddress()
	case "Address.streetName":
		return m.faker.Address().StreetName()
	/*
		BOOLEANS
	*/
	case "Boolean.boolean":
		return strconv.FormatBool(m.faker.Boolean().Bool())
	case "Boolean.booleanWithChance":
		chance, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return strconv.FormatBool(m.faker.Boolean().Bool())
		}
		return strconv.FormatBool(m.faker.Boolean().BoolWithChance(chance))
	/*
		CAR
	*/
	case "Car.maker":
		return m.faker.Car().Maker()
	case "Car.model":
		return m.faker.Car().Model()
	case "Car.plate":
		return m.faker.Car().Plate()
	/*
		COMPANY
	*/
	case "Company.name":
		return m.faker.Company().Name()
	case "Company.suffix":
		return m.faker.Company().Suffix()
	case "Company.cnpj":
		cnpj, err := regen.Generate("[0-9]{2}.[0-9]{3}.[0-9]{3}/[0-9]{4}-[0-9]{2}")
		if err != nil {
			log.Fatalf("Failed to generate CNPJ: %v\n", err)
		}
		return cnpj
	/*
		CURRENCY
	*/
	case "Currency.currencyCode":
		return m.faker.Currency().Code()
	case "Currency.currencyContry":
		return m.faker.Currency().Country()
	case "Currency.currencyName":
		return m.faker.Currency().Currency()
	case "Currency.currencyNumber":
		return strconv.Itoa(m.faker.Currency().Number())
	/*
		FILE
	*/
	case "File.filenameWithExtension":
		return m.faker.File().FilenameWithExtension()
	case "File.extension":
		return m.faker.File().Extension()
	/*
		INTERNET
	*/
	case "Internet.email":
		return m.faker.Internet().Email()
	case "Internet.domain":
		return m.faker.Internet().Domain()
	case "Internet.url":
		return m.faker.Internet().URL()
	case "Internet.ipv4":
		return m.faker.Internet().Ipv4()
	case "Internet.password":
		return m.faker.Internet().Password()
	/*
		LOREM
	*/
	case "Lorem.paragraph":
		sentences, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return m.faker.Lorem().Paragraph(1)
		}
		return m.faker.Lorem().Paragraph(sentences)
	case "Lorem.paragraphs":
		paragraphs, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return strings.Join(m.faker.Lorem().Paragraphs(1), "")
		}
		return strings.Join(m.faker.Lorem().Paragraphs(paragraphs), "\n")
	case "Lorem.sentence":
		words, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return m.faker.Lorem().Sentence(1)
		}
		return m.faker.Lorem().Sentence(words)
	case "Lorem.sentences":
		sentences, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return strings.Join(m.faker.Lorem().Sentences(1), "")
		}
		return strings.Join(m.faker.Lorem().Sentences(sentences), "\n")
	case "Lorem.word":
		return m.faker.Lorem().Word()
	case "Lorem.words":
		words, err := strconv.Atoi(functionParams[0])
		if err != nil {
			return strings.Join(m.faker.Lorem().Words(1), " ")
		}
		return strings.Join(m.faker.Lorem().Words(words), "")
	/*
		PAYMENT
	*/
	case "Payment.creditCardExpirationDate":
		return m.faker.Payment().CreditCardExpirationDateString()
	case "Payment.creditCardNumber":
		return m.faker.Payment().CreditCardNumber()
	case "Payment.creditCardType":
		return m.faker.Payment().CreditCardType()
	case "Payment.creditCardCvv":
		cvv, err := regen.Generate("[0-9]{3}")
		if err != nil {
			log.Fatalf("Failed to generate CVV: %v\n", err)
		}
		return cvv
	/*
		PERSON
	*/
	case "Person.phoneNumber":
		return m.faker.Person().Contact().Phone
	case "Person.email":
		return m.faker.Person().Contact().Email
	case "Person.firstName":
		return m.faker.Person().FirstName()
	case "Person.lastName":
		return m.faker.Person().LastName()
	case "Person.fullName":
		return m.faker.Person().FirstName() + " " + m.faker.Person().LastName()
	case "Person.cpf":
		cpf, err := regen.Generate("[0-9]{3}.[0-9]{3}.[0-9]{3}-[0-9]{2}")
		if err != nil {
			log.Fatalf("Failed to generate CPF: %v\n", err)
		}
		return cpf
	/*
		TIME
	*/

	/*
		UUID
	*/
	case "UUID.uuidv4":
		return m.faker.UUID().V4()
	/*
		USER AGENT
	*/
	case "UserAgent.userAgent":
		return m.faker.UserAgent().UserAgent()
	default:
		log.Fatalf("Unknown mock function: %v\n", mockFunction)
		return ""
	}
}
