package main

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = shim.NewLogger("repo")

const CIBContractNumVer string = "1.0.0"
const CIBContractBuild string = "15"
const CIBContractVersion string = CIBContractNumVer + " build: " + CIBContractBuild + " (19.11.2018)"
const dateFormatInVal string = "02.01.2006 15:04:05 -0700"
const dateFormat string = "02.01.2006 15:04:05"
const dateFormatDM string = "02.01.2006"
const dateFormatM string = "01.2006"

// contract.IntMeth = 365/366 // стандартный для рублей метод расчета процентов = 365/366
// 1)	 [число дней в году ( «базы» как делителя для ставки)] = факт. числу дней в текущем году, т.е. 365.
// 2)	 [количество дней в периоде расчета] = Дата конца периода расчета – Дата начала периода расчета
const countDaysYear = 365

type CIBContract struct {
}

type contractType struct {
	DealNum            string                 `json:"dealNum"`            //Номер сделки присваивается в НРД автоматически
	DealDate           string                 `json:"dealDate"`           //Дата заключения сделки
	Leg1Date           string                 `json:"leg1Date"`           //Дата расчетов первой части РЕПО
	Leg2Date           string                 `json:"leg2Date"`           //Дата расчетов второй части РЕПО
	Amount             float64                `json:"amount"`             //Сумма сделки в валюте сделки
	Currency           string                 `json:"currency"`           //Валюта сделки
	Leg1DealType       string                 `json:"leg1DealType"`       //Способ расчета по первой части РЕПО: DVP1; DVP3
	Threshold1         float64                `json:"threshold1"`         //Нижний порог переоценки
	Threshold2         float64                `json:"threshold2"`         //Верхний порог переоценки
	RepoRate           float64                `json:"repoRate"`           //Ставка	Фиксированная ставка (% годовых)
	IntMeth            string                 `json:"intMeth"`            //Код метод расчета процентов. Cтандартный для рублей метод расчета процентов = 365/366
	CollateralReceiver collateralReceiverType `json:"collateralReceiver"` //Блок описания Кредитора по сделки
	CollateralGiver    collateralGiverType    `json:"collateralGiver"`    //Блок описания Кредитора по сделки
	MasterAgreement    masterAgreementType    `json:"masterAgreement"`    //Блок описания Генерального соглашения
	Collateral         collateralType         `json:"collateral"`         //Блок описания Генерального соглашения
	SuoParams          suoParamsType          `json:"suoParams"`          //Блок описания дополнительных параметров СУО НРД
	History            []historyType          `json:"history"`            //Блок истории по контракту
	ContrSigned        contrSignedType        `json:"contrSigned"`        //Блок подписи контракта
	ObligationList     []obligationListType   `json:"obligationList"`     //Блок всех обязательств по сделке и статус их исполнения
	Status             int                    `json:"status"`             //Статус контрактаЖ 0 - новый, 1 - подтверждён (в работе), 2 - закрыт
}

type contrSignedType struct {
	SellerContrSigned signedDetailType `json:"sellerContrSigned"`
	BuyerContrSigned  signedDetailType `json:"buyerContrSigned"`
}

type signedDetailType struct {
	DateTime           string `json:"dateTime"`
	WhoSigned          string `json:"whoSigned"`
	Text               string `json:"text"`
	TextSigned         string `json:"textSigned"`
	SignatureAlgorithm string `json:"signatureAlgorithm"`
	PublicKey          string `json:"publicKey"`
	Confirmation       int    `json:"confirmation"`
}

type obligationListType struct {
	CommitmentID       string  `json:"commitmentID"`
	DateTime           string  `json:"dateTime"`
	WhoMadeChanges     string  `json:"whoMadeChanges"`
	TextDescription    string  `json:"textDescription"`
	Amount             float64 `json:"amount"`
	QuantitySecurities int     `json:"quantitySecurities"`
	PerformanceStatus  int     `json:"performanceStatus"` // 0 - исполнено, 1- не исполнено, 2- отменено
}

type historyType struct {
	DateTimeChange string `json:"dateTimeChange"`
	WhatHasChanged string `json:"whatHasChanged"`
	WhoMadeChanges string `json:"whoMadeChanges"`
	Detailing      string `json:"detailing"`
}

type collateralReceiverType struct {
	Code          string  `json:"code"`          //Уникальный код организации в НРД
	ShortName     string  `json:"shortName"`     //Краткое наименование организации
	TypeOfTrade   string  `json:"typeOfTrade"`   //Тип владения активами: S - собственные активы; D - доверительное управление; L - активы клиента, брокер
	DepoSectionID float64 `json:"depoSectionID"` //Идентификатор раздела счета депо
	DepoAcc       string  `json:"depoAcc"`       //Номер счета депо
	DepoSection   string  `json:"depoSection"`   //Номер раздела счета депо
	Account       string  `json:"account"`       //Номер торгового банковского счета
}

type collateralGiverType struct {
	Code          string  `json:"code"`          //Код НРД	Уникальный код организации в НРД
	ShortName     string  `json:"shortName"`     //Краткое наименование организации
	TypeOfTrade   string  `json:"typeOfTrade"`   //Тип владения активами: S - собственные активы; D - доверительное управление; L - активы клиента, брокер
	DepoSectionID float64 `json:"depoSectionID"` //Идентификатор раздела счета депо
	DepoAcc       string  `json:"depoAcc"`       //Номер счета депо
	DepoSection   string  `json:"depoSection"`   //Номер раздела счета депо
	Account       string  `json:"account"`       //Номер торгового банковского счета
}

type masterAgreementType struct {
	Code string `json:"code"` //Код ГС в Репозитарии НРД
	Date string `json:"date"` //Дата заключения ГС
}

type collateralType struct {
	SecurityCode       string  `json:"securityCode"`       //Код НРД ценной бумаги
	SecurityIsin       string  `json:"securityIsin"`       //Код ISIN ценной бумаги
	SecurityName       string  `json:"securityName"`       //Краткое наименование ценной бумаги
	Quantity           int     `json:"quantity"`           //Количество ценных бумаг в обеспечении
	Discount           float64 `json:"discount"`           //Дисконт (в %)
	PriceTypesPriority string  `json:"priceTypesPriority"` //Список кодов НРД источников цен для переоценки
}

type suoParamsType struct {
	Reuse         string `json:"reuse"`         //Реюз обеспечения: Y – разрешен; N – запрещен
	ReturnVar     string `json:"returnVar"`     //Описание варианта возврата доходов по РЕПО
	ShiftTermDate string `json:"shiftTermDate"` // Досрочное исполнение	Условие изменения досрочного исполнения сделки
	AutoMargin    string `json:"autoMargin"`    // Автоматическое маржирование	Признак автомаржирования в СУО НРД: Y – разрешено; N – запрещено
}

func main() {
	err := shim.Start(new(CIBContract))
	if err != nil {
		logger.Errorf("Error starting CIBContract: %s", err)
	}
}

func (t *CIBContract) Init(APIstub shim.ChaincodeStubInterface) pb.Response {
	logger.Infof("CIBContract version : %v", CIBContractVersion)
	return shim.Success([]byte("Version chaincode: " + CIBContractVersion))
}

func (t *CIBContract) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	shim.LogLevel("DEBUG")
	logger.Infof("CIBContract version : %v", CIBContractVersion)
	function, args := stub.GetFunctionAndParameters()
	logger.Infof("invoke is running %v", function)
	if function == "addContract"+"_"+CIBContractNumVer {
		return t.addContract(stub, args)
	} 

	return shim.Error("Received unknown function invocation")
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func roundFloat(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

func (t *CIBContract) addContract(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	var amount, threshold1, threshold2, repoRate, discount float64
	var quantity int

	if len(args) != 30 {
		return pb.Response{Status: 403, Message: "Incorrect number of arguments. Expecting 30"}
	}
	if len(args[0]) <= 0 {
		return pb.Response{Status: 403, Message: "DealNum must be a non-empty"}
	}

	dealNum := strings.TrimSpace(args[0])
	dealDate := strings.TrimSpace(args[1])
	leg1Date := strings.TrimSpace(args[2])
	leg2Date := strings.TrimSpace(args[3])

	amount, err = strconv.ParseFloat(strings.TrimSpace(args[4]), 64)
	if err != nil {
		return pb.Response{Status: 403, Message: "Expected float value for amount"}
	}

	currency := strings.TrimSpace(args[5])
	leg1DealType := strings.TrimSpace(args[6])

	threshold1, err = strconv.ParseFloat(strings.TrimSpace(args[7]), 64)
	if err != nil {
		return pb.Response{Status: 403, Message: "Expected float value for threshold1"}
	}

	threshold2, err = strconv.ParseFloat(strings.TrimSpace(args[8]), 64)
	if err != nil {
		return pb.Response{Status: 403, Message: "Expected float value for threshold2"}
	}

	repoRate, err = strconv.ParseFloat(strings.TrimSpace(args[9]), 64)
	if err != nil {
		return pb.Response{Status: 403, Message: "Expected float value for repoRate"}
	}

	intMeth := strings.TrimSpace(args[10])
	// intMeth, err := strconv.Atoi(strings.TrimSpace(args[10]))
	// if err != nil {
	// 	return pb.Response{Status: 403, Message: "Expected integer value for intMeth"}
	// }

	collateralReceiver := collateralReceiverType{}
	collateralReceiver.Code = strings.TrimSpace(args[11])
	collateralReceiver.ShortName = strings.TrimSpace(args[12])
	collateralReceiver.TypeOfTrade = "" //strings.TrimSpace(args[13])

	// depoSectionID, err = strconv.ParseFloat(strings.TrimSpace(args[14]), 64)
	// if err != nil {
	// 	return pb.Response{Status: 403, Message: "Expected float value for depoSectionID"}
	// }

	collateralReceiver.DepoSectionID = float64(0)

	collateralReceiver.DepoAcc = ""     //strings.TrimSpace(args[15])
	collateralReceiver.DepoSection = "" //strings.TrimSpace(args[16])
	collateralReceiver.Account = ""     //strings.TrimSpace(args[17])

	collateralGiver := collateralGiverType{}
	collateralGiver.Code = strings.TrimSpace(args[13])
	collateralGiver.ShortName = strings.TrimSpace(args[14])
	// collateralGiver.TypeOfTrade = strings.TrimSpace(args[20])

	// depoSectionIDG, err = strconv.ParseFloat(strings.TrimSpace(args[21]), 64)
	// if err != nil {
	// 	return pb.Response{Status: 403, Message: "Expected float value for depoSectionIDG"}
	// }

	collateralGiver.DepoSectionID = float64(0)

	collateralGiver.DepoAcc = ""     //strings.TrimSpace(args[22])
	collateralGiver.DepoSection = "" //strings.TrimSpace(args[23])
	collateralGiver.Account = ""     //strings.TrimSpace(args[24])

	masterAgreement := masterAgreementType{}
	masterAgreement.Code = strings.TrimSpace(args[15])
	masterAgreement.Date = strings.TrimSpace(args[16])

	collateral := collateralType{}
	collateral.SecurityCode = strings.TrimSpace(args[17])
	collateral.SecurityIsin = strings.TrimSpace(args[18])
	collateral.SecurityName = strings.TrimSpace(args[19])

	quantity, err = strconv.Atoi(strings.TrimSpace(args[20]))
	if err != nil {
		return pb.Response{Status: 403, Message: "Expected integer value for quantity securities"}
	}

	collateral.Quantity = quantity

	discount, err = strconv.ParseFloat(strings.TrimSpace(args[21]), 64)
	if err != nil {
		return pb.Response{Status: 403, Message: "Expected float value for discount"}
	}

	collateral.Discount = discount
	collateral.PriceTypesPriority = strings.TrimSpace(args[22])

	suoParams := suoParamsType{}
	suoParams.Reuse = strings.TrimSpace(args[23])
	suoParams.ReturnVar = strings.TrimSpace(args[24])
	suoParams.ShiftTermDate = strings.TrimSpace(args[25])
	suoParams.AutoMargin = strings.TrimSpace(args[26])

	history := []historyType{}
	ht := historyType{dealDate, "Create Contract", strings.TrimSpace(args[27]), "Contract " + dealNum + " successfully created"}
	history = append(history, ht)

	contrSigned := contrSignedType{}
	contrSigned.SellerContrSigned = signedDetailType{}
	contrSigned.SellerContrSigned.TextSigned = ""
	contrSigned.BuyerContrSigned = signedDetailType{}
	contrSigned.BuyerContrSigned.TextSigned = ""

	obligation := []obligationListType{}
	obl := obligationListType{strings.TrimSpace(args[28]), leg1Date, strings.TrimSpace(args[27]), "bank to client", (-1) * amount, 0, 1}
	obligation = append(obligation, obl)

	obl = obligationListType{strings.TrimSpace(args[29]), leg1Date, strings.TrimSpace(args[27]), "client to bank", 0, quantity, 1}
	obligation = append(obligation, obl)

	contract := &contractType{dealNum, dealDate, leg1Date, leg2Date, amount, currency, leg1DealType, threshold1, threshold2, repoRate, intMeth, collateralReceiver, collateralGiver, masterAgreement, collateral, suoParams, history, contrSigned, obligation, 0}

	contractJSONasBytes, err := json.Marshal(&contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(dealNum, contractJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = t.createIndex(stub, string("allContr"), []string{"allContr", contract.DealNum})
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte("Contract " + contract.DealNum + " successfully created"))

}

func (t *CIBContract) createIndex(stub shim.ChaincodeStubInterface, indexName string, indexKey []string) error {
	compositeKey, err := stub.CreateCompositeKey(indexName, indexKey)
	if err != nil {
		return err
	}
	value := []byte{0x00}
	stub.PutState(compositeKey, value)
	return nil
}
