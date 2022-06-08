//fixed
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

type DispatchDetails struct {
	ItemID string `json:"item_id"`

	DeliveredQuantity string `json:"delivered_quantity"`
	BalanceQuantity   string `json:"balance_quantity"`
	DispatchId        string `json:"dispatch_id"`
	DispatchDate      string `json:"dispatch_date"`
	DispatchQuantity  string `json:"dispatch_quantity"`
	RelatedDetid        string `json:"related_detid"`

	//Green Coffee Info Section-Done--------------------------

	ExpectedComp []ExpectedPOComp       `json:"expected_composition"`
	VendorComp   []VendorDispatchedComp `json:"vendor_composition"`
}

type ExpectedPOComp struct {
	Density       string `json:"density"`
	Moisture      string `json:"moisture"`
	Browns        string `json:"browns"`
	Blacks        string `json:"blacks"`
	BrokenBits    string `json:"brokenbits"`
	InsectedBeans string `json:"insectedbeans"`
	Bleached      string `json:"bleached"`
	Husk          string `json:"husk"`
	Sticks        string `json:"sticks"`
	Stones        string `json:"stones"`
	BeansRetained string `json:"beansretained"`
}
type VendorDispatchedComp struct {
	Density       string `json:"density"`
	Moisture      string `json:"moisture"`
	Browns        string `json:"browns"`
	Blacks        string `json:"blacks"`
	BrokenBits    string `json:"brokenbits"`
	InsectedBeans string `json:"insectedbeans"`
	Bleached      string `json:"bleached"`
	Husk          string `json:"husk"`
	Sticks        string `json:"sticks"`
	Stones        string `json:"stones"`
	BeansRetained string `json:"beansretained"`
}

func viewGCPODispatchDetails(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var dd DispatchDetails

	err := json.Unmarshal([]byte(request.Body), &dd)
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
	// var rows *sql.Rows

	if dd.DispatchId != "" {
		//---------------------Fetch Single Dispatch Info-------------------------------------//

		log.Println("Fetching Single/Multiple Dispatch Information the Contract #")
		var itemid,quantity,dispatch_date,delivered_quantity,balance_quantity,parent_detid sql.NullString
		sqlStatementMDInfo1 := `select d.itemid,d.quantity,d.dispatch_date,
							m.delivered_quantity, (m.expected_quantity-m.delivered_quantity) as balance_quantity,
							d.parent_detid
							from dbo.pur_gc_po_dispatch_master_newpg d
							left join dbo.inv_gc_po_mrin_master_newpg as m on m.detid=d.detid
							where d.detid=$1`
		rows9, err9 := db.Query(sqlStatementMDInfo1, dd.DispatchId)
		log.Println("Dispatch Info Fetch Query Executed")
		if err9 != nil {
			log.Println("Dispatch Info Fetch Query failed")
			log.Println(err9.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		for rows9.Next() {

			err9 = rows9.Scan(&itemid,&quantity,&dispatch_date,&delivered_quantity,&balance_quantity,&parent_detid)
		}
				
		dd.ItemID=itemid.String
		dd.DispatchQuantity=quantity.String
		dd.DispatchDate=dispatch_date.String
		dd.DeliveredQuantity=delivered_quantity.String
		dd.BalanceQuantity=balance_quantity.String
		dd.RelatedDetid=parent_detid.String
		
		//---------------------Fetch Expected Dispatch Comp-------------------------------------//
		log.Println("Fetching Audit Log Info #")
		sqlStatementEC := `SELECT density, moisture, browns, blacks, brokenbits, insectedbeans, 
				bleached, husk, sticks, stones, beansretained
					FROM dbo.pur_gc_po_composition_master_newpg 
					where itemid=$1`
		rowsEC, errEC := db.Query(sqlStatementEC, dd.ItemID)
		log.Println("Audit Info Fetch Query Executed")
		if errEC != nil {
			log.Println("Audit Info Fetch Query failed")
			log.Println(errEC.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errEC.Error(), false}, nil
		}

		for rowsEC.Next() {
			var edc ExpectedPOComp
			errEC = rowsEC.Scan(&edc.Density, &edc.Moisture, &edc.Browns, &edc.Blacks, &edc.BrokenBits,
				&edc.InsectedBeans, &edc.Bleached, &edc.Husk, &edc.Sticks, &edc.Stones,
				&edc.BeansRetained)
			expCompDetails := append(dd.ExpectedComp, edc)
			dd.ExpectedComp = expCompDetails
			log.Println("added one")

		}

		//---------------------Fetch Vendor Dispatch Comp-------------------------------------//
		log.Println("Fetching Audit Log Info #")
		sqlStatementVC := `SELECT density, moisture, browns, blacks, brokenbits, insectedbeans, 
						bleached, husk, sticks, stones, beansretained
							FROM dbo.pur_gc_po_composition_vendor_newpg 
							where detid=$1`
		rowsVC, errVC := db.Query(sqlStatementVC, dd.DispatchId)
		log.Println("Audit Info Fetch Query Executed")
		if errVC != nil {
			log.Println("Audit Info Fetch Query failed")
			log.Println(err9.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errVC.Error(), false}, nil
		}

		for rowsVC.Next() {
			var vdc VendorDispatchedComp
			errVC = rowsVC.Scan(&vdc.Density, &vdc.Moisture, &vdc.Browns, &vdc.Blacks, &vdc.BrokenBits,
				&vdc.InsectedBeans, &vdc.Bleached, &vdc.Husk, &vdc.Sticks, &vdc.Stones,
				&vdc.BeansRetained)
			venCompDetails := append(dd.VendorComp, vdc)
			dd.VendorComp = venCompDetails
			log.Println("added one")

		}

		res, _ := json.Marshal(dd)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil

	} else {
		return events.APIGatewayProxyResponse{500, headers, nil, string("Missing Dispatch Id"), false}, nil
	}

}
func main() {
	lambda.Start(viewGCPODispatchDetails)
}
