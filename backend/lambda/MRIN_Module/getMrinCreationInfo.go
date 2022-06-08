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

type Entity struct {
	EntityId   string `json:"entity_id"`
	EntityName string `json:"entity_name"`
}

type PoDetail struct {
	PoId          string `json:"po_id"`
	PoNo          string `json:"po_no"`
	VendorId      string `json:"vendor_id"`
	PurchasePrice string `json:"purchase_price"`
}

type DispatchDetail struct {
	DispatchId        string `json:"dispatch_id"`
	DispatchDate      string `json:"dispatch_date"`
	Quantity          string `json:"quantity"`
	RelatedDispatchId string `json:"relateddispatch_id"`
	Mrinno            string `json:"mrinno"`
	BalanceQuantity   string `json:"balance_quantity"`
	IsMrinCreated     bool   `json:"ismrin_created"`
	InvoiceNo         string `json:"invoice_no"`
	VehicleNo         string `json:"vehicle_no"`
	VendorSubmit      bool   `json:"vendor_submitted"`
}

type Input struct {
	Type     string `json:"type"`
	EntityId string `json:"entity_id"`
	PoNo     string `json:"po_no"`
}

func getMrinCreationInfo(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var input Input
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
	if input.Type == "allEntities" {
		log.Println("get all entities", input.Type)
		sqlStatement := `select entityid, entityname from dbo.PROJECT_ENTITY_MASTER`
		rows, err = db.Query(sqlStatement)

		var allEntities []Entity
		defer rows.Close()
		for rows.Next() {
			var et Entity
			err = rows.Scan(&et.EntityId, &et.EntityName)
			allEntities = append(allEntities, et)
		}

		res, _ = json.Marshal(allEntities)
	} else if input.Type == "poidsOnEntityId" {
		log.Println("get all poids on entity id", input.Type)
		sqlStatement := `select poid, pono, vendorid 
						from dbo.pur_gc_po_con_master_newpg 
						where status=3 and delivery_at_id=$1`
		rows, err = db.Query(sqlStatement, input.EntityId)

		var purchasePrice sql.NullString
		var allPoDetails []PoDetail
		defer rows.Close()
		for rows.Next() {
			var po PoDetail
			err = rows.Scan(&po.PoId, &po.PoNo, &po.VendorId)
			if po.PoId != "" {
				sqlStatementD1 := `select purchase_price from dbo.pur_gc_po_details_newpg where poid=$1 limit 1`

				rows2, errD2 := db.Query(sqlStatementD1, po.PoId)
				if errD2 != nil {
					log.Println("Fetching Purchase price value to auto populate has failed", errD2.Error())
				}
				for rows2.Next() {
					err1 := rows2.Scan(&purchasePrice)
					if err1 != nil {
						log.Println("Purchase price value to auto populate has failed", err1.Error())
					}
				}
			}
			po.PurchasePrice = purchasePrice.String
			allPoDetails = append(allPoDetails, po)
		}

		res, _ = json.Marshal(allPoDetails)
	} else if input.Type == "dispatchesonpono" {
		log.Println("get all poids on entity id", input.Type)
		sqlStatementD1 := `select d.detid, d.quantity, d.dispatch_date, (m.expected_quantity-m.delivered_quantity) as balance_quantity, d.parent_detid, m.mrinno
									from dbo.pur_gc_po_dispatch_master_newpg d
									left join dbo.inv_gc_po_mrin_master_newpg as m on m.detid=d.detid 
									where d.pono=$1`

		rowsD1, errD1 := db.Query(sqlStatementD1, input.PoNo)
		if errD1 != nil {
			log.Println("Fetching Dispatch Details from DB failed")
			log.Println(errD1.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errD1.Error(), false}, nil
		}

		var allDispatchDetails []DispatchDetail
		var dispatchDate, relatedDispatchId, mrinno, balanceQuantity, invoiceNo, vehicleNo sql.NullString
		defer rowsD1.Close()
		for rowsD1.Next() {
			var ds DispatchDetail
			errD1 = rowsD1.Scan(&ds.DispatchId, &ds.Quantity, &dispatchDate, &balanceQuantity, &relatedDispatchId, &mrinno)
			log.Println("Get invoice detail for ", ds.DispatchId)
			if ds.DispatchId != "" {
				sqlStatementD1 := `select invoiceno,vehicle_no from dbo.pur_gc_po_composition_vendor_newpg where detid=$1 and status='Submitted' limit 1`

				rows2, errD2 := db.Query(sqlStatementD1, ds.DispatchId)
				if errD2 != nil {
					log.Println("Fetching Dispatch Details from DB failed to populate invoice and vehicle no", errD2.Error())
				}
				for rows2.Next() {
					err1 := rows2.Scan(&invoiceNo, &vehicleNo)
					if err1 != nil {
						log.Println("Scanningh Dispatch Details from DB failed to populate invoice and vehicle no", err1.Error())
					}
					ds.VendorSubmit = true
				}
			}

			ds.DispatchDate = dispatchDate.String
			ds.RelatedDispatchId = relatedDispatchId.String
			ds.BalanceQuantity = balanceQuantity.String
			ds.InvoiceNo = invoiceNo.String
			ds.VehicleNo = vehicleNo.String
			ds.IsMrinCreated = mrinno.Valid
			allDispatchDetails = append(allDispatchDetails, ds)
			log.Println("Dispatch Scanned is: ", &ds.DispatchId)
		}

		res, _ = json.Marshal(allDispatchDetails)
	}

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getMrinCreationInfo)
}
