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

type InputDetails struct {
	AccountID      string `json:"accountid"`
	UserID         string `json:"userid"`
	LoggedInUserId string `json:"loggedinuserid"`
}

func reassignAccount(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	var inputDetails InputDetails

	err := json.Unmarshal([]byte(request.Body), &inputDetails)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	defer db.Close()

	// check db
	err = db.Ping()
	var role string
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	var rows1 *sql.Rows
	sqlStatementAUser1 := `SELECT role FROM dbo.users_master_newpg
							where userid=$1`
	rows1, err = db.Query(sqlStatementAUser1, inputDetails.LoggedInUserId)
	for rows1.Next() {
		err = rows1.Scan(&role)
	}

	var rows2 *sql.Rows
	var uname string
	sqlStatementAUser2 := `select username from dbo.users_master_newpg where userid=$1`
	rows2, err = db.Query(sqlStatementAUser2, inputDetails.UserID)
	for rows2.Next() {
		err = rows2.Scan(&uname)
	}

	//var rows *sql.Rows
	fmt.Println("Connected!")
	fmt.Println(role)
	if role == "Managing Director" {

		sqlStatementre1 := `UPDATE dbo.accounts_master SET 		
						account_owner = $1				 
						where accountid=$2`

		_, err = db.Query(sqlStatementre1, uname, inputDetails.AccountID)
		//defer rows1.Close()

		res, _ := json.Marshal("Reassign Successful.")
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	} else if role != "Managing Director" {
		res, _ := json.Marshal("Permission denied, you dont have the permission.")
		return events.APIGatewayProxyResponse{402, headers, nil, string(res), false}, nil
	}
	res, _ := json.Marshal("Success.")
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(reassignAccount)
}
