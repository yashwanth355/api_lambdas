// working code-tested
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	cognito "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	_ "github.com/lib/pq"
)

const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Ccl_RDS_DB#2022"
	dbname   = "ccldevdb"
)

type App struct {
	CognitoClient *cognito.CognitoIdentityProvider
	UserPoolID    string
	AppClientID   string
}

type Credentials struct {
	Username         string `json:"username"`
	Previouspassword string `json:"previouspassword,omitempty"`
	Newpassword      string `json:"newpassword,omitempty"`
}

func changePassword(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

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
	var rows *sql.Rows
	var credentials Credentials
	err1 := json.Unmarshal([]byte(request.Body), &credentials)
	if err1 != nil {
		log.Println(err1)
		return events.APIGatewayProxyResponse{500, headers, nil, string(""), false}, nil
	}

	var oldPassword sql.NullString
	log.Println(credentials.Username)
	sqlStatement1 := `select u.password from dbo.users_master_newpg u where u.emailid=$1`
	rows, err = db.Query(sqlStatement1, credentials.Username)
	if err != nil {
		log.Println("Unable to get password from database", err.Error())
	}
	for rows.Next() {
		err = rows.Scan(&oldPassword)
	}

	if oldPassword.String != credentials.Previouspassword {
		log.Println("Previous password is not matched: oldPassword", oldPassword.String, "Given :", credentials.Previouspassword)
		return events.APIGatewayProxyResponse{400, headers, nil, string("previous password is not matched"), false}, nil
	} else {
		log.Println("Updating the password in cognito")
		mySession := session.Must(session.NewSession())
		cognitoRegion := os.Getenv("AWS_COGNITO_REGION")
		cognitoUserPoolId := os.Getenv("COGNITO_USER_POOL_ID")
		cognitoAppClientId := os.Getenv("COGNITO_APP_CLIENT_ID")

		svc := cognitoidentityprovider.New(mySession, aws.NewConfig().WithRegion(cognitoRegion))

		cognitoClient := App{
			CognitoClient: svc,
			UserPoolID:    cognitoUserPoolId,
			AppClientID:   cognitoAppClientId,
		}

		changePassword := &cognito.AdminSetUserPasswordInput{
			Password:   aws.String(credentials.Newpassword),
			Permanent:  aws.Bool(true),
			UserPoolId: aws.String(cognitoUserPoolId),
			Username:   aws.String(credentials.Username),
		}

		_, err = cognitoClient.CognitoClient.AdminSetUserPassword(changePassword)
		if err != nil {
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, string("Failed to change password"), false}, nil
		}

		sqlStatement1 := `update dbo.users_master_newpg set password=$1 where emailid=$2`
		_, err = db.Query(sqlStatement1, credentials.Newpassword, credentials.Username)
		if err != nil {
			log.Println("Unable to update password in database", err.Error())
		}

		return events.APIGatewayProxyResponse{200, headers, nil, string("Successfully changed password"), false}, nil
	}
}

func main() {
	lambda.Start(changePassword)
}
