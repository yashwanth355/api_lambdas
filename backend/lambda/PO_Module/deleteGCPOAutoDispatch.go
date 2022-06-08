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

type ItemDispatch struct {
	ItemID string `json:"item_id"`
	PoNO            string `json:"po_no"`
	DispatchID       string `json:"dispatch_id"`
	DispatchQuantity string `json:"dispatch_quantity"`
	DispatchDate     string `json:"dispatch_date"`
	
}

func deleteGCPOAutoDispatch(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var dd ItemDispatch

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

	if dd.DispatchID != "" {
		//---------------------Update balance quantity of PO-------------------------------------//
		sqlStatementMDInfo1 := `update dbo.pur_gc_po_details_newpg
								set
								balance_quantity=balance_quantity+(select quantity from dbo.pur_gc_po_dispatch_master_newpg where detid=$1)
								where pono=$2`
		_,errMDInfo1 := db.Query(sqlStatementMDInfo1, dd.DispatchID,dd.PoNO)
		if errMDInfo1 != nil {
			log.Println("Dispatch update failed")
			log.Println(errMDInfo1.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errMDInfo1.Error(), false}, nil
		}	
		log.Println("Entered delete dispatch module")
		log.Println("Date & DD",dd.DispatchID)
		sqlStatementMDInfo2 := `delete from dbo.pur_gc_po_dispatch_master_newpg
								where detid=$1`
								log.Println(sqlStatementMDInfo2)
		_, errMDInfo2 := db.Query(sqlStatementMDInfo2, dd.DispatchID)
		log.Println("Dispatch Info Fetch Query Executed")
		if errMDInfo2 != nil {
			log.Println("Dispatch update failed")
			log.Println(errMDInfo2.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errMDInfo2.Error(), false}, nil
		}	
		
		return events.APIGatewayProxyResponse{200, headers, nil, string("Dispatch deleted & balance updated to PO successfully"), false}, nil

	} else {
		return events.APIGatewayProxyResponse{500, headers, nil, string("Missing Dispatch Id"), false}, nil
	}

}
func main() {
	lambda.Start(deleteGCPOAutoDispatch)
}
