package main

import (
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

type App struct {
	CognitoClient *cognito.CognitoIdentityProvider
	UserPoolID    string
	AppClientID   string
}

type Credentials struct {
	Username string `json:"username"`
}

func forgotPassword(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	var credentials Credentials
	err := json.Unmarshal([]byte(request.Body), &credentials)
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
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

	forgotPassword := &cognito.ForgotPasswordInput{
		ClientId: aws.String(cognitoClient.AppClientID),
		Username: aws.String(credentials.Username),
	}
	response, err := cognitoClient.CognitoClient.ForgotPassword(forgotPassword)
	if err != nil {
		log.Println(err.Error())
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	return events.APIGatewayProxyResponse{200, headers, nil, response.String(), false}, nil
}

func main() {
	lambda.Start(forgotPassword)
}
