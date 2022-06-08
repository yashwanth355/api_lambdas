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
	DSNo             string `json:"number"`
	DDate            string `json:"date"`
}

func editGCPODispatchDetails(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
		//---------------------Fetch Single Dispatch Info-------------------------------------//

		log.Println("Entered edit dispatch module")
		log.Println("Date & DD",dd.DispatchDate,dd.DispatchID)
		sqlStatementMDInfo1 := `update dbo.pur_gc_po_dispatch_master_newpg
								set
								dispatch_date=$1
								where detid=$2`
		log.Println(sqlStatementMDInfo1)					
		_, err9 := db.Query(sqlStatementMDInfo1, dd.DispatchDate,dd.DispatchID)
		log.Println("Dispatch Info Fetch Query Executed")
		if err9 != nil {
			log.Println("Dispatch update failed")
			log.Println(err9.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}	
		
		return events.APIGatewayProxyResponse{200, headers, nil, string("Dispatch updated successfully"), false}, nil

	} else {
		return events.APIGatewayProxyResponse{500, headers, nil, string("Missing Dispatch Id"), false}, nil
	}

}
func main() {
	lambda.Start(editGCPODispatchDetails)
}
