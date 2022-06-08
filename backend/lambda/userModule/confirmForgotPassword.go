package main

import (
	"database/sql"

	"fmt"

	_ "github.com/lib/pq"

	//SES

	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	cognito "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
)

const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Ccl_RDS_DB#2022"
	dbname   = "ccldevdb"
	//email-SMTP
	from_email = "itsupport@continental.coffee"
	smtp_pass  = "is@98765"
	// smtp server configuration.
	smtpHost = "smtp.gmail.com"
	smtpPort = "587"
)

type App struct {
	CognitoClient *cognito.CognitoIdentityProvider
	UserPoolID    string
	AppClientID   string
}

type Credentials struct {
	Username         string `json:"username"`
	Password         string `json:"password,omitempty"`
	Confirmationcode string `json:"confirmationcode,omitempty"`
}

func ConfirmForgotPassword(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var credentials Credentials
	err := json.Unmarshal([]byte(request.Body), &credentials)
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

	confirmForgotPassword := &cognito.ConfirmForgotPasswordInput{
		ClientId:         aws.String(cognitoClient.AppClientID),
		Username:         aws.String(credentials.Username),
		ConfirmationCode: aws.String(credentials.Confirmationcode),
		Password:         aws.String(credentials.Password),
	}

	response, err := cognitoClient.CognitoClient.ConfirmForgotPassword(confirmForgotPassword)
	if err != nil {
		log.Println(err.Error())
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	//Update Pwd In DB in User Table

	sqlStatementUDB1 := `UPDATE dbo.users_master_newpg
					   		SET 
					   	password=$1
					   where emailid=$2`

	_, err = db.Query(sqlStatementUDB1, credentials.Password, credentials.Username)

	if err != nil {
		log.Println(err.Error())
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	return events.APIGatewayProxyResponse{200, headers, nil, response.String(), false}, nil
}

func main() {
	lambda.Start(ConfirmForgotPassword)
}
