//Deployed with pwd change
//Deployed with table name changed to dbo
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	cognito "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	_ "github.com/lib/pq"
)

const (
	host         = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port         = 5432
	user         = "postgres"
	password     = "Ccl_RDS_DB#2022"
	dbname       = "ccldevdb"
	flowUserType = "USER_PASSWORD_AUTH"
)

type App struct {
	CognitoClient *cognito.CognitoIdentityProvider
	UserPoolID    string
	AppClientID   string
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type Result struct {
	Id           *string `json:"id"`
	Role         string  `json:"role"`
	UserName     string  `json:"user_name"`
	Userid       string  `json:"user_id"`
	ExpiryIn     *int64  `json:"expiryin"`
	RefreshToken string  `json:"refreshtoken"`
}

func login(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
		return events.APIGatewayProxyResponse{500, headers, nil, err1.Error(), false}, nil
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
	flow := aws.String(flowUserType)
	params := map[string]*string{
		"USERNAME": aws.String(credentials.Username),
		"PASSWORD": aws.String(credentials.Password),
	}
	authTry := &cognito.InitiateAuthInput{
		AuthFlow:       flow,
		AuthParameters: params,
		ClientId:       aws.String(cognitoClient.AppClientID),
	}
	response, err := cognitoClient.CognitoClient.InitiateAuth(authTry)
	if err != nil {
		log.Println(err.Error())
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	var loginData Result
	loginData.Id = response.AuthenticationResult.AccessToken
	loginData.ExpiryIn = response.AuthenticationResult.ExpiresIn
	loginData.RefreshToken = *response.AuthenticationResult.RefreshToken

	if credentials.Username != "" {

		log.Println(credentials.Username)
		sqlStatement1 := `select u.role,u.userid,u.username from dbo.users_master_newpg u where u.emailid=$1`
		rows, err = db.Query(sqlStatement1, credentials.Username)

		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&loginData.Role, &loginData.Userid, &loginData.UserName)
		}

	} else {
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	log.Println(loginData)
	res, _ := json.Marshal(loginData)
	log.Println(res)
	// log user connect
	//Find latest idsno
	var lastidsno int
	sqlStatementUID1 := `SELECT idsno 
						FROM dbo.auditlog_users_login_newpg 
						where idsno is not null
						ORDER BY idsno DESC 
						LIMIT 1`
	rows, err = db.Query(sqlStatementUID1)
	if err != nil {
		log.Println(err.Error())
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	for rows.Next() {
		err = rows.Scan(&lastidsno)
	}
	log.Println("Last Idsno: ", lastidsno)
	//Generating UserIds NOs----------------
	idsno := lastidsno + 1
	logid := "Logid-" + strconv.Itoa(idsno)
	loggedintime := time.Now()
	log.Println("Generated logid : ", logid)
	log.Println("Logged in time ", loggedintime)
	sqlStatementIU1 := `INSERT INTO dbo.auditlog_users_login_newpg(
						idsno,logid, loggedintime, userid, firstname, middlename,
						lastname, emailid, alias, username, empcode, designation, 
						company, department, role, division, employee, profile, 
						delegatedapprover, manager, password, ext, phone, mobile, 
						country, state, street, city, postalcode, active, title, 
						crm_legacyid, erp_legacyid)SELECT $1,$2,$3,userid, firstname, 
						middlename, lastname, emailid, alias, username, empcode, designation, 
						company, department, role, division, employee, profile, delegatedapprover, 
						manager, password, ext, phone, mobile, country, state, street, city, 
						postalcode, active, title, crm_legacyid, erp_legacyid
						FROM dbo.users_master_newpg where emailid=$4`

	_, err = db.Query(sqlStatementIU1, idsno, logid, loggedintime, credentials.Username)

	if err != nil {
		log.Println(err.Error())
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
}
func main() {
	lambda.Start(login)
}
