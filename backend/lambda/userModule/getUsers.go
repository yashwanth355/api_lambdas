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

type Users struct {
	Userid         string `json:"userid"`
	UserName       string `json:"username"`
	Type           string `json:"type"`
	LoggedInUserID string `json:"loginuserid"`
}

func getUsers(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	var user Users

	err = json.Unmarshal([]byte(request.Body), &user)

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
	if user.Type == "Leads" {
		sqlStatement := `SELECT userid, username FROM dbo.users_master_newpg where role='Marketing Executive'`
		rows, err := db.Query(sqlStatement)
		if err != nil {
			log.Println(err)
			log.Println("unable to get users list")
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		var userDetailsList = make([]Users, 0)
		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&user.Userid, &user.UserName)
			userDetailsList = append(userDetailsList, user)
			log.Println(userDetailsList)
		}

		res, _ := json.Marshal(userDetailsList)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil

	} else if user.Type == "Accounts" {
		sqlStatement := `SELECT userid, username FROM dbo.users_master_newpg where role='Marketing Executive'`
		rows, err := db.Query(sqlStatement)

		var userDetailsList = make([]Users, 0)
		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&user.Userid, &user.UserName)
			userDetailsList = append(userDetailsList, user)
			log.Println(userDetailsList)
		}

		if err != nil {
			log.Println(err)
			log.Println("unable to get users list")
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}

		res, _ := json.Marshal(userDetailsList)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil

	}

	return events.APIGatewayProxyResponse{200, headers, nil, string("Success"), false}, nil
}

func main() {
	lambda.Start(getUsers)
}
