package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	//test comments
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

type Accounts struct {
	AccountID      int    `json:"accountid"`
	AccountName    string `json:"accountname"`
	Aliases        string `json:"aliases"`
	AccountType    string `json:"accounttypeid"`
	AccountOwner   string `json:"account_owner"`
	ApprovalStatus string `json:"masterstatus"`
}
type Input struct {
	Filter string `json:"filter"`
}

func getAllAccounts(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	var param string
	if input.Filter == "" {
		param = " ORDER BY accidsno DESC"
	} else {
		param = "where " + input.Filter + " ORDER BY createddate DESC"
	}

	log.Println("filter Query :", param)
	var rows *sql.Rows
	sqlStatement1 := fmt.Sprintf("SELECT accountid,accountname,aliases,accounttypeid,account_owner,masterstatus FROM dbo.AccountsGrid %s", param)
	rows, err = db.Query(sqlStatement1)
	var allAccounts []Accounts
	var accountName, aliases, accountType, accountOwner, approvalStatus sql.NullString
	defer rows.Close()
	for rows.Next() {
		var account Accounts
		err = rows.Scan(&account.AccountID, &accountName, &aliases, &accountType, &accountOwner, &approvalStatus)
		account.AccountName = accountName.String
		account.Aliases = aliases.String
		account.AccountType = accountType.String
		account.AccountOwner = accountOwner.String
		account.ApprovalStatus = approvalStatus.String
		var accounttypes []string
		if account.AccountType != "" {
			z := strings.Split(account.AccountType, ",")
			for i, z := range z {
				log.Println("get account name", i, z)
				sqlStatement1 := `SELECT accounttype FROM dbo.cms_account_type_master where accounttypeid=$1`
				rows1, err1 := db.Query(sqlStatement1, z)

				if err1 != nil {
					log.Println(err, "Not able to Add account type")
				}

				for rows1.Next() {
					var accounttype string
					err = rows1.Scan(&accounttype)
					accounttypes = append(accounttypes, accounttype)
				}
			}
		}
		account.AccountType = strings.Join(accounttypes, ",")
		allAccounts = append(allAccounts, account)
	}

	res, _ := json.Marshal(allAccounts)
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}

func main() {
	lambda.Start(getAllAccounts)
}
