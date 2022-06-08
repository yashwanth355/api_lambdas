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

type User struct {
	Email      string `json:"emailid"`
	Password   string `json:"password,omitempty"`
	FirstName  string `json:"firstname"`
	LastName   string `json:"lastname"`
	MiddleName string `json:"middlename"`
	Alias      string `json:"alias"`
	UserName   string `json:"username"`
	Title      string `json:"title"`
	Department string `json:"department"`
	Role       string `json:"role"`
	Profile    string `json:"profile"`
}

func registration(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	var user User
	err := json.Unmarshal([]byte(request.Body), &user)
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

	createUser := &cognito.SignUpInput{
		Username: aws.String(user.Email),
		Password: aws.String(user.Password),
		ClientId: aws.String(cognitoClient.AppClientID),
		UserAttributes: []*cognito.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(user.Email),
			},
			{
				Name:  aws.String("family_name"),
				Value: aws.String(user.Alias),
			},
			{
				Name:  aws.String("given_name"),
				Value: aws.String(user.UserName),
			},
			{
				Name:  aws.String("custom:title"),
				Value: aws.String(user.Title),
			},
			{
				Name:  aws.String("custom:department"),
				Value: aws.String(user.Department),
			},
			{
				Name:  aws.String("custom:role"),
				Value: aws.String(user.Role),
			},
			{
				Name:  aws.String("profile"),
				Value: aws.String(user.Profile),
			},
		},
	}

	_, err = cognitoClient.CognitoClient.SignUp(createUser)
	if err != nil {
		log.Println(err.Error())
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	return events.APIGatewayProxyResponse{200, headers, nil, "Successfully created", false}, nil
}

func main() {
	lambda.Start(registration)
}
