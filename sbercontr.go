//Copyright Sberbank 2018 All Rights Reserved
//Project: REPO.
//Project Customer: CIB
//Developer: Baykov D.V.

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
	} else if function == "contractVersion"+"_"+CIBContractNumVer {
		return shim.Success([]byte("Version chaincode: " + CIBContractVersion))
	} else if function == "viewContract"+"_"+CIBContractNumVer {
		return t.viewContract(stub, args[0])
	} else if function == "getBlockHistory"+"_"+CIBContractNumVer {
		return t.getBlockHistory(stub, args[0])
	} else if function == "viewListContracts"+"_"+CIBContractNumVer {
		return t.viewListContracts(stub, args)
	} else if function == "addHistoryToContract"+"_"+CIBContractNumVer {
		return t.addHistoryToContract(stub, args)
	} else if function == "getHistoryFromContract"+"_"+CIBContractNumVer {
		return t.getHistoryFromContract(stub, args[0])
	} else if function == "addSignToContract"+"_"+CIBContractNumVer {
		return t.addSignToContract(stub, args)
	} else if function == "getObligationList"+"_"+CIBContractNumVer {
		return t.getObligationList(stub, args[0])
	} else if function == "addObligationStatus"+"_"+CIBContractNumVer {
		return t.addObligationStatus(stub, args)
	} else if function == "updContract"+"_"+CIBContractNumVer {
		return t.updContract(stub, args)
	} else if function == "addPriceToContract"+"_"+CIBContractNumVer {
		return t.addPriceToContract(stub, args)
	} else if function == "acceptContract"+"_"+CIBContractNumVer {
		return t.acceptContract(stub, args)
	} else if function == "closeContract"+"_"+CIBContractNumVer {
		return t.closeContract(stub, args)
	} else if function == "getTestContract"+"_"+CIBContractNumVer {
		return t.getTestContract(stub)
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

	// dealNum - args[0]
	// dealDate - args[1]
	// leg1Date - args[2]
	// leg2Date - args[3]
	// amount - args[4]
	// currency - args[5]
	// leg1DealType - args[6]
	// threshold1 - args[7]
	// threshold2 - args[8]
	// repoRate - args[9]
	// intMeth - args[10]
	// collateralReceiver.Code - args[11]
	// collateralReceiver.ShortName - args[12]
	// collateralGiver.Code - args[13]
	// collateralGiver.ShortName - args[14]
	// masterAgreement.Code - args[15]
	// masterAgreement.Date - args[16]
	// collateral.SecurityCode - args[17]
	// collateral.SecurityIsin - args[18]
	// collateral.SecurityName - args[19]
	// collateral.Quantity - args[20]
	// collateral.Discount - args[21]
	// collateral.PriceTypesPriority - args[22]
	// suoParams.Reuse - args[23]
	// suoParams.ReturnVar - args[24]
	// suoParams.ShiftTermDate - args[25]
	// suoParams.AutoMargin - args[26]
	// WhoMadeChanges - args[27] (only history)
	// CommitmentID1 - args[28]
	// CommitmentID2 - args[29]

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

func (t *CIBContract) getTestContract(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success([]byte("Hello, world!"))
}

func (t *CIBContract) viewContract(stub shim.ChaincodeStubInterface, contrID string) pb.Response {
	var err error
	var valBytes []byte

	valBytes, err = stub.GetState(contrID)

	if err != nil {
		return pb.Response{Status: 403, Message: "Failed to get contract " + contrID}
	} else if valBytes == nil {
		return pb.Response{Status: 404, Message: "Contract does not exist: " + contrID}
	}

	return shim.Success(valBytes)
}

func (t *CIBContract) getBlockHistory(stub shim.ChaincodeStubInterface, contrID string) pb.Response {
	type AuditHistory struct {
		TxId   string       `json:"txId"`
		TxDate string       `json:"txDate"`
		Value  contractType `json:"value"`
	}
	var history []AuditHistory
	var contract contractType

	if len(contrID) <= 0 {
		return shim.Error("ContrID must be a non-empty string")
	}

	resultsIterator, err := stub.GetHistoryForKey(contrID)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	for resultsIterator.HasNext() {
		historyData, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		var tx AuditHistory
		tx.TxId = historyData.TxId
		tx.TxDate = time.Unix(historyData.Timestamp.Seconds, int64(historyData.Timestamp.Nanos)).String()
		json.Unmarshal(historyData.Value, &contract)
		if historyData.Value == nil {
			var emptyContractType contractType
			tx.Value = emptyContractType
		} else {
			json.Unmarshal(historyData.Value, &contract)
			tx.Value = contract
		}
		history = append(history, tx)
	}

	historyAsBytes, _ := json.Marshal(history)
	return shim.Success(historyAsBytes)
}

func (t *CIBContract) viewListContracts(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	var indexName, res, tmpText string
	var retVal shim.StateQueryIteratorInterface

	indexName = args[0]

	if len(args) == 2 {
		retVal, err = stub.GetStateByPartialCompositeKey(indexName, []string{args[1]})
	} else if len(args) == 3 {
		retVal, err = stub.GetStateByPartialCompositeKey(indexName, []string{args[1], args[2]})
	} else if len(args) == 4 {
		retVal, err = stub.GetStateByPartialCompositeKey(indexName, []string{args[1], args[2], args[3]})
	} else {
		return pb.Response{Status: 403, Message: "Arguments not equal function"}
	}

	if err != nil {
		return pb.Response{Status: 403, Message: "Failed to get list contracts, index: " + indexName}
	} else if retVal == nil {
		return pb.Response{Status: 404, Message: "List contracts does not exist, index:  " + indexName}
	}

	defer retVal.Close()

	var i int
	res = "["
	for i = 0; retVal.HasNext(); i++ {
		indexKey, err := retVal.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		_, compositeKeyParts, err := stub.SplitCompositeKey(indexKey.GetKey())
		if err != nil {
			return shim.Error(err.Error())
		}

		if len(args) == 2 {
			tmpText = string(t.viewContract(stub, compositeKeyParts[1]).Payload)
		} else if len(args) == 3 {
			tmpText = string(t.viewContract(stub, compositeKeyParts[2]).Payload)
		} else if len(args) == 4 {
			tmpText = string(t.viewContract(stub, compositeKeyParts[3]).Payload)
		}

		if i == 0 {
			res = res + tmpText
		} else {
			res = res + "," + tmpText
		}

	}

	res = res + "]"
	return shim.Success([]byte(res))
}

func (t *CIBContract) closeContract(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	// contrID - args[0]
	// CommitmentID1 - args[1]
	// CommitmentID2 - args[2]
	// date - args[3]
	// WhoMadeChanges - args[4] (only history)

	if len(args) != 5 {
		return pb.Response{Status: 403, Message: "Incorrect number of arguments. Expecting 5"}
	}

	contrID := strings.TrimSpace(args[0])
	valBytes, err := stub.GetState(contrID)

	if err != nil {
		return pb.Response{Status: 403, Message: "Failed to get contract " + contrID}
	} else if valBytes == nil {
		return pb.Response{Status: 404, Message: "Contract does not exist: " + contrID}
	}

	var contract contractType
	err = json.Unmarshal(valBytes, &contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	if contract.Status == 1 {
		signingDate, _ := time.Parse(dateFormatDM, contract.Leg1Date[:10])
		endDateCur, _ := time.Parse(dateFormatDM, strings.TrimSpace(args[3])[:10])
		deltaSigningDate := endDateCur.Sub(signingDate)
		rePurchasePriceCur := roundFloat((contract.Amount + (contract.Amount*contract.RepoRate*float64(int(deltaSigningDate.Hours()/24))/float64(countDaysYear))/100), 2)

		obl := obligationListType{strings.TrimSpace(args[1]), strings.TrimSpace(args[3]), strings.TrimSpace(args[4]), "client to bank (close contarct)", rePurchasePriceCur, 0, 1}
		contract.ObligationList = append(contract.ObligationList, obl)

		obl = obligationListType{strings.TrimSpace(args[2]), strings.TrimSpace(args[3]), strings.TrimSpace(args[4]), "bank to client (close contarct)", 0, (-1) * contract.Collateral.Quantity, 1}
		contract.ObligationList = append(contract.ObligationList, obl)

		contract.Status = 2

		ht := historyType{strings.TrimSpace(args[3]), "close contract", strings.TrimSpace(args[3]), "close contract"}
		contract.History = append(contract.History, ht)

		contractJSONasBytes, err := json.Marshal(&contract)
		if err != nil {
			return shim.Error(err.Error())
		}

		err = stub.PutState(contrID, contractJSONasBytes)
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success([]byte("Contract " + contrID + " successfully updated (add history)"))
	} else {
		return pb.Response{Status: 403, Message: "Contract : " + contrID + " not closed, because it not confirmed"}
	}
}

func (t *CIBContract) addHistoryToContract(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	if len(args) != 5 {
		return pb.Response{Status: 403, Message: "Incorrect number of arguments. Expecting 5"}
	}

	contrID := strings.TrimSpace(args[0])
	valBytes, err := stub.GetState(contrID)

	if err != nil {
		return pb.Response{Status: 403, Message: "Failed to get contract " + contrID}
	} else if valBytes == nil {
		return pb.Response{Status: 404, Message: "Contract does not exist: " + contrID}
	}

	var contract contractType
	err = json.Unmarshal(valBytes, &contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	ht := historyType{strings.TrimSpace(args[1]), strings.TrimSpace(args[2]), strings.TrimSpace(args[3]), strings.TrimSpace(args[4])}
	contract.History = append(contract.History, ht)

	contractJSONasBytes, err := json.Marshal(&contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(contrID, contractJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte("Contract " + contrID + " successfully updated (add history)"))
}

func (t *CIBContract) addObligationStatus(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	if len(args) != 6 {
		return pb.Response{Status: 403, Message: "Incorrect number of arguments. Expecting 6"}
	}

	contrID := strings.TrimSpace(args[0])
	valBytes, err := stub.GetState(contrID)

	if err != nil {
		return pb.Response{Status: 403, Message: "Failed to get contract " + contrID}
	} else if valBytes == nil {
		return pb.Response{Status: 404, Message: "Contract does not exist: " + contrID}
	}

	var contract contractType
	err = json.Unmarshal(valBytes, &contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	for i, obligation := range contract.ObligationList {
		if obligation.CommitmentID == strings.TrimSpace(args[1]) {
			contract.ObligationList[i].DateTime = strings.TrimSpace(args[2])
			contract.ObligationList[i].WhoMadeChanges = strings.TrimSpace(args[3])
			contract.ObligationList[i].TextDescription = strings.TrimSpace(args[4])
			performanceStatus, err := strconv.Atoi(strings.TrimSpace(args[5]))
			if err != nil {
				return pb.Response{Status: 403, Message: "Expected integer value for performance status"}
			}
			contract.ObligationList[i].PerformanceStatus = performanceStatus
			break
		}
	}

	ht := historyType{strings.TrimSpace(args[2]), "Add obligation status", strings.TrimSpace(args[3]), strings.TrimSpace(args[4])}
	contract.History = append(contract.History, ht)

	contractJSONasBytes, err := json.Marshal(&contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(contrID, contractJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte("Contract " + contrID + " successfully updated (add obligation status)"))
}

func (t *CIBContract) updContract(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	if len(args) != 5 {
		return pb.Response{Status: 403, Message: "Incorrect number of arguments. Expecting 5"}
	}

	contrID := strings.TrimSpace(args[0])
	valBytes, err := stub.GetState(contrID)

	if err != nil {
		return pb.Response{Status: 403, Message: "Failed to get contract " + contrID}
	} else if valBytes == nil {
		return pb.Response{Status: 404, Message: "Contract does not exist: " + contrID}
	}

	var contract contractType
	err = json.Unmarshal(valBytes, &contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	if contract.Status == 1 {
		if strings.TrimSpace(args[4]) != "0" {
			quantitySecurities, err := strconv.Atoi(strings.TrimSpace(args[4]))
			if err != nil {
				return pb.Response{Status: 403, Message: "Expected integer value for quantity securities"}
			}
			contract.Collateral.Quantity = contract.Collateral.Quantity + quantitySecurities
		}

		ht := historyType{strings.TrimSpace(args[1]), "Update Contract (mount(obligation) or add quantity securities to Contract)", strings.TrimSpace(args[2]), "amount: " + strings.TrimSpace(args[3]) + ", quantity securities: " + strings.TrimSpace(args[4])}
		contract.History = append(contract.History, ht)

		contractJSONasBytes, err := json.Marshal(&contract)
		if err != nil {
			return shim.Error(err.Error())
		}

		err = stub.PutState(contrID, contractJSONasBytes)
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success([]byte("Contract " + contrID + " successfully updated (amount(obligation) or add quantity securities to Contract)"))
	} else {
		return pb.Response{Status: 403, Message: "Contract : " + contrID + " not updated, because it not confirmed"}
	}
}

func (t *CIBContract) addPriceToContract(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	// var dateTimeChange, whoMadeChanges, securityCode, securityIsin, securityName, marketPriceType, marketPriceCur, priceDate, incomeCur, priceCur, oblID string
	// var marketPrice, couponIncome, price, necesSUM, allSumCollateral, rePurchasePriceCur float64

	if len(args) != 15 {
		return pb.Response{Status: 403, Message: "Incorrect number of arguments. Expecting 15"}
	}

	dateTimeChange := strings.TrimSpace(args[1])
	whoMadeChanges := strings.TrimSpace(args[2])
	// securityCode := strings.TrimSpace(args[3])
	// securityIsin := strings.TrimSpace(args[4])
	// securityName := strings.TrimSpace(args[5])

	// marketPrice, err := strconv.ParseFloat(strings.TrimSpace(args[6]), 64)
	// if err != nil {
	// 	return pb.Response{Status: 403, Message: "Expected float value for market price"}
	// }

	// marketPriceType := strings.TrimSpace(args[7])
	// marketPriceCur := strings.TrimSpace(args[8])
	// priceDate := strings.TrimSpace(args[9])
	// couponIncome, err := strconv.ParseFloat(strings.TrimSpace(args[10]), 64)
	// if err != nil {
	// 	return pb.Response{Status: 403, Message: "Expected float value for coupon income"}
	// }
	// incomeCur := strings.TrimSpace(args[11])
	price, err := strconv.ParseFloat(strings.TrimSpace(args[12]), 64)
	if err != nil {
		return pb.Response{Status: 403, Message: "Expected float value for price"}
	}
	// priceCur := strings.TrimSpace(args[13])
	oblID := strings.TrimSpace(args[14])

	contrID := strings.TrimSpace(args[0])
	valBytes, err := stub.GetState(contrID)

	if err != nil {
		return pb.Response{Status: 403, Message: "Failed to get contract " + contrID}
	} else if valBytes == nil {
		return pb.Response{Status: 404, Message: "Contract does not exist: " + contrID}
	}

	var contract contractType
	err = json.Unmarshal(valBytes, &contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	if contract.Status == 1 {

		allSumCollateral := float64(0)
		rePurchasePriceCur := float64(0)
		necesSUM := float64(0)
		priceIzm := price * (float64(1) - contract.Collateral.Discount/100)

		signingDate, _ := time.Parse(dateFormatDM, contract.Leg1Date[:10])
		endDateCur, _ := time.Parse(dateFormatDM, dateTimeChange[:10])
		deltaSigningDate := endDateCur.Sub(signingDate)
		rePurchasePriceCur = roundFloat((contract.Amount + (contract.Amount*contract.RepoRate*float64(int(deltaSigningDate.Hours()/24))/float64(countDaysYear))/100), 2)

		if priceIzm*float64(contract.Collateral.Quantity) <= rePurchasePriceCur-(contract.Threshold1*rePurchasePriceCur)/100 {
			allSecurities := math.Ceil(rePurchasePriceCur / priceIzm)
			necesSecurities := math.Ceil(allSecurities - float64(contract.Collateral.Quantity))
			obl := obligationListType{oblID, dateTimeChange, whoMadeChanges, "need add securities from client", 0, int(necesSecurities), 1}
			contract.ObligationList = append(contract.ObligationList, obl)
		}

		if priceIzm*float64(contract.Collateral.Quantity) > rePurchasePriceCur+(contract.Threshold2*rePurchasePriceCur)/100 {
			allSecurities := math.Ceil(rePurchasePriceCur / priceIzm)
			necesSecurities := math.Ceil(allSecurities - float64(contract.Collateral.Quantity))
			obl := obligationListType{oblID, dateTimeChange, whoMadeChanges, "need get securities from bank to client", 0, int(necesSecurities), 1}
			contract.ObligationList = append(contract.ObligationList, obl)
		}

		ht := historyType{strings.TrimSpace(args[1]), "Add new price to Contract", strings.TrimSpace(args[2]), "new price: " + strings.TrimSpace(args[12])}
		contract.History = append(contract.History, ht)

		contractJSONasBytes, err := json.Marshal(&contract)
		if err != nil {
			return shim.Error(err.Error())
		}

		err = stub.PutState(contrID, contractJSONasBytes)
		if err != nil {
			return shim.Error(err.Error())
		}

		return shim.Success([]byte("{\"dealNum\":\"" + contract.DealNum + "\",\"necesSUM\":\"" + strconv.FormatFloat(float64(necesSUM), 'f', 2, 64) + "\",\"rePurchasePriceCur\":\"" + strconv.FormatFloat(float64(rePurchasePriceCur), 'f', 2, 64) + "\",\"allSumCollateral\":\"" + strconv.FormatFloat(float64(allSumCollateral), 'f', 0, 64) + "\"}"))
	} else {
		return shim.Success([]byte("{\"dealNum\":\"" + contract.DealNum + "\",\"necesSUM\":\"0\",\"rePurchasePriceCur\":\"0\",\"allSumCollateral\":\"0\"}"))
	}
}

func (t *CIBContract) getObligationList(stub shim.ChaincodeStubInterface, contrID string) pb.Response {
	var err error
	var valBytes []byte

	if len(contrID) == 0 {
		return pb.Response{Status: 403, Message: "Incorrect number of arguments. Expecting 1"}
	}

	valBytes, err = stub.GetState(contrID)

	if err != nil {
		return pb.Response{Status: 403, Message: "Failed to get contract " + contrID}
	} else if valBytes == nil {
		return pb.Response{Status: 404, Message: "Contract does not exist: " + contrID}
	}

	var contract contractType
	err = json.Unmarshal(valBytes, &contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	ol := contract.ObligationList
	contractJSONasBytes, err := json.Marshal(&ol)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(contractJSONasBytes)
}

func (t *CIBContract) getHistoryFromContract(stub shim.ChaincodeStubInterface, contrID string) pb.Response {
	var err error
	var valBytes []byte

	if len(contrID) == 0 {
		return pb.Response{Status: 403, Message: "Incorrect number of arguments. Expecting 1"}
	}

	valBytes, err = stub.GetState(contrID)

	if err != nil {
		return pb.Response{Status: 403, Message: "Failed to get contract " + contrID}
	} else if valBytes == nil {
		return pb.Response{Status: 404, Message: "Contract does not exist: " + contrID}
	}

	var contract contractType
	err = json.Unmarshal(valBytes, &contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	ht := contract.History
	contractJSONasBytes, err := json.Marshal(&ht)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(contractJSONasBytes)
}

func (t *CIBContract) acceptContract(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	if len(args) != 4 {
		return pb.Response{Status: 403, Message: "Incorrect number of arguments. Expecting 4"}
	}

	// contrID -- args[0]
	// type Org -- args[1]
	// DateTime -- args[2]
	// WhoMadeChanges -- args[3]

	contrID := strings.TrimSpace(args[0])
	valBytes, err := stub.GetState(contrID)

	if err != nil {
		return pb.Response{Status: 403, Message: "Failed to get contract " + contrID}
	} else if valBytes == nil {
		return pb.Response{Status: 404, Message: "Contract does not exist: " + contrID}
	}

	var contract contractType
	err = json.Unmarshal(valBytes, &contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	if strings.TrimSpace(args[1]) == "buyer" {
		contract.ContrSigned.BuyerContrSigned.Confirmation = 1
	}

	if strings.TrimSpace(args[1]) == "seller" {
		contract.ContrSigned.SellerContrSigned.Confirmation = 1
	}

	if contract.ContrSigned.BuyerContrSigned.Confirmation == 1 && contract.ContrSigned.SellerContrSigned.Confirmation == 1 {
		contract.Status = 1
	}

	ht := historyType{strings.TrimSpace(args[2]), "Contract accepted", strings.TrimSpace(args[3]), "Contract accepted type Org: " + strings.TrimSpace(args[1]) + ", date: " + strings.TrimSpace(args[2])}
	contract.History = append(contract.History, ht)

	contractJSONasBytes, err := json.Marshal(&contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(contrID, contractJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte("Contract " + contrID + " was successfully accepted"))
}

func (t *CIBContract) addSignToContract(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	if len(args) != 8 {
		return pb.Response{Status: 403, Message: "Incorrect number of arguments. Expecting 8"}
	}
	// if len(args[0]) <= 0 {
	// 	return pb.Response{Status: 403, Message: "Contract ID must be a non-empty"}
	// }
	// if len(args[1]) <= 0 {
	// 	return pb.Response{Status: 403, Message: "Date time must be a non-empty"}
	// }
	// if len(args[2]) <= 0 {
	// 	return pb.Response{Status: 403, Message: "Who accepted must be a non-empty"}
	// }
	// if len(args[3]) <= 0 {
	// 	return pb.Response{Status: 403, Message: "Type Org must be a non-empty"}
	// }
	// if len(args[4]) <= 0 {
	// 	return pb.Response{Status: 403, Message: "Text signed must be a non-empty"}
	// }
	// if len(args[5]) <= 0 {
	// 	return pb.Response{Status: 403, Message: "Signature algorithm must be a non-empty"}
	// }
	// if len(args[6]) <= 0 {
	// 	return pb.Response{Status: 403, Message: "Public key must be a non-empty"}
	// }
	// if len(args[7]) <= 0 {
	// 	return pb.Response{Status: 403, Message: "Text must be a non-empty"}
	// }

	contrID := strings.TrimSpace(args[0])
	valBytes, err := stub.GetState(contrID)

	if err != nil {
		return pb.Response{Status: 403, Message: "Failed to get contract " + contrID}
	} else if valBytes == nil {
		return pb.Response{Status: 404, Message: "Contract does not exist: " + contrID}
	}

	var contract contractType
	err = json.Unmarshal(valBytes, &contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	contrSigned := contract.ContrSigned.SellerContrSigned

	if strings.TrimSpace(args[3]) == "buyer" {
		contrSigned = contract.ContrSigned.BuyerContrSigned

	}

	if contrSigned.TextSigned != "" {
		return pb.Response{Status: 403, Message: "Contract " + contrID + " already signed"}
	}

	contrSigned.DateTime = strings.TrimSpace(args[1])
	contrSigned.PublicKey = strings.TrimSpace(args[6])
	contrSigned.SignatureAlgorithm = strings.TrimSpace(args[5])
	contrSigned.Text = strings.TrimSpace(args[7])
	contrSigned.TextSigned = strings.TrimSpace(args[4])
	contrSigned.WhoSigned = strings.TrimSpace(args[2])

	if strings.TrimSpace(args[3]) == "buyer" {
		contract.ContrSigned.BuyerContrSigned = contrSigned
	}

	if strings.TrimSpace(args[3]) == "seller" {
		contract.ContrSigned.SellerContrSigned = contrSigned
	}

	ht := historyType{strings.TrimSpace(args[1]), "Contract signing", strings.TrimSpace(args[2]), "Contract signing type Org: " + strings.TrimSpace(args[3]) + ", date: " + strings.TrimSpace(args[1]) + ", text:" + strings.TrimSpace(args[7]) + ", textSigned:" + strings.TrimSpace(args[4]) + ", signatureAlgorithm:" + strings.TrimSpace(args[5]) + ", publicKey:" + strings.TrimSpace(args[6])}
	contract.History = append(contract.History, ht)

	contractJSONasBytes, err := json.Marshal(&contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(contrID, contractJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte("Contract " + contrID + " was successfully signed"))
}
