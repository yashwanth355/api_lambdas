package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
)

const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Ccl_RDS_DB#2022"
	dbname   = "ccldevdb"
)

type MrinList struct {
	Type              string     `json:"type"`
	PoNo              string     `json:"po_no"`
	MrinId            NullString `json:"mrin_id"`
	Mrindate          NullString `json:"mrindate"`
	ExpectedQuantity  NullString `json:"expected_quantity"`
	DeliveredQuantity string     `json:"delivered_quantity"`
	BalanceQuantity   string     `json:"balance_quantity"`
	DispatchId        string     `json:"dispatch_id"`
	RelatedDetid      NullString `json:"related_detid"`
	QCStatus          string     `json:"qcStatus"`
	APStatus          string     `json:"apStatus"`
	//Green Coffee Info Section-Done--------------------------

	ItemID string `json:"item_id"`
}

type NullString struct {
	sql.NullString
}

// MarshalJSON for NullString
func (ns *NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ns.String)
}

func getMrinListForPoView(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var input MrinList
	err := json.Unmarshal([]byte(request.Body), &input)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	defer db.Close()

	// check db
	err = db.Ping()

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	fmt.Println("Connected!")

	var res []byte
	var rows *sql.Rows
	if input.Type == "mrinsonponoforview" {
		log.Println("get all mrins on pono", input.Type)
		var apstatus, qcstatus sql.NullString
		sqlStatement := `select mrinid,TO_CHAR(mrindate,'DD-MON-YY') as mrindate, detid, expected_quantity, delivered_quantity,
							(expected_quantity-delivered_quantity) as balance_quantity,
							related_detid,ap_status, qc_status
							from dbo.inv_gc_po_mrin_master_newpg 
							where 
							pono=$1`
		rows, err = db.Query(sqlStatement, input.PoNo)

		var allMrins []MrinList
		defer rows.Close()
		for rows.Next() {
			var mr MrinList
			err = rows.Scan(&mr.MrinId, &mr.Mrindate, &mr.DispatchId, &mr.ExpectedQuantity, &mr.DeliveredQuantity, &mr.BalanceQuantity, &mr.RelatedDetid, &apstatus, &qcstatus)
			mr.QCStatus = qcstatus.String
			mr.APStatus = apstatus.String
			allMrins = append(allMrins, mr)
		}

		res, _ = json.Marshal(allMrins)
	} else if input.Type == "viewmrinsondispatch" {
		log.Println("get all mrins on pono", input.Type)
		sqlStatementMD1 := `select mrinid,TO_CHAR(mrindate,'DD-MON-YY') as mrindate, detid, expected_quantity, delivered_quantity,
							(expected_quantity-delivered_quantity) as balance_quantity,
							related_detid,ap_status, qc_status
							from dbo.inv_gc_po_mrin_master_newpg 
							where 
							detid=$1`
		rowsMD1, errMD1 := db.Query(sqlStatementMD1, input.DispatchId)
		if errMD1 != nil {
			log.Println("Fetching Dispatch Details from DB failed")
			log.Println(errMD1.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errMD1.Error(), false}, nil
		}
		var allMrins []MrinList
		defer rowsMD1.Close()
		for rowsMD1.Next() {
			var mr MrinList
			errMD1 = rowsMD1.Scan(&mr.MrinId, &mr.Mrindate, &mr.DispatchId, &mr.ExpectedQuantity, &mr.DeliveredQuantity, &mr.BalanceQuantity, &mr.RelatedDetid, &mr.APStatus, &mr.QCStatus)
			allMrins = append(allMrins, mr)
		}
		// log.Println("The GC Composition for the Dispatch #")
		// sqlStatementPOGC1:=`SELECT density, moisture, browns, blacks, brokenbits, insectedbeans, bleached, husk, sticks, stones, beansretained
		// 							FROM dbo.pur_gc_po_composition_vendor_newpg
		// 							where detid=$1`
		// rows7, err7 := db.Query(sqlStatementPOGC1,input.DispatchId)
		// log.Println("GC Fetch Query Executed")
		// if err7 != nil {
		// 	log.Println("Fetching GC Composition Details from DB failed")
		// 	log.Println(err7.Error())
		// 	return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		// 	}

		// for rows7.Next() {
		// 	var mr MrinList
		// 	err7 = rows7.Scan(&mr.Density,&mr.Moisture,&mr.Browns,&mr.Blacks,&mr.BrokenBits,&mr.InsectedBeans,&mr.Bleached,
		// 						&mr.Husk,&mr.Sticks,&mr.Stones,&mr.BeansRetained)

		// }

		res, _ = json.Marshal(allMrins)
	}

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getMrinListForPoView)
}
