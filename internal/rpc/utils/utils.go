package rpcutils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	teaconst "github.com/TeaOSLab/EdgeAPI/internal/const"
	"github.com/TeaOSLab/EdgeAPI/internal/db/models"
	"github.com/TeaOSLab/EdgeAPI/internal/encrypt"
	"github.com/TeaOSLab/EdgeAPI/internal/utils"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/maps"
	"google.golang.org/grpc/metadata"
	"time"
)

type UserType = string

const (
	UserTypeNone     = "none"
	UserTypeAdmin    = "admin"
	UserTypeUser     = "user"
	UserTypeProvider = "provider"
	UserTypeNode     = "node"
	UserTypeMonitor  = "monitor"
	UserTypeStat     = "stat"
	UserTypeDNS      = "dns"
	UserTypeLog      = "log"
	UserTypeAPI      = "api"
)

// 校验请求
func ValidateRequest(ctx context.Context, userTypes ...UserType) (userType UserType, userId int64, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return UserTypeNone, 0, errors.New("context: need 'nodeId'")
	}
	nodeIds := md.Get("nodeid")
	if len(nodeIds) == 0 || len(nodeIds[0]) == 0 {
		return UserTypeNone, 0, errors.New("context: need 'nodeId'")
	}
	nodeId := nodeIds[0]

	// 获取角色Node信息
	// TODO 缓存节点ID相关信息
	apiToken, err := models.SharedApiTokenDAO.FindEnabledTokenWithNode(nodeId)
	if err != nil {
		utils.PrintError(err)
		return UserTypeNone, 0, err
	}
	nodeUserId := int64(0)
	if apiToken == nil {
		return UserTypeNode, 0, errors.New("context: invalid api token")
	}

	tokens := md.Get("token")
	if len(tokens) == 0 || len(tokens[0]) == 0 {
		return UserTypeNone, 0, errors.New("context: need 'token'")
	}
	token := tokens[0]

	data, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return UserTypeNone, 0, err
	}

	method, err := encrypt.NewMethodInstance(teaconst.EncryptMethod, apiToken.Secret, nodeId)
	if err != nil {
		utils.PrintError(err)
		return UserTypeNone, 0, err
	}
	data, err = method.Decrypt(data)
	if err != nil {
		return UserTypeNone, 0, err
	}
	if len(data) == 0 {
		return UserTypeNone, 0, errors.New("invalid token")
	}

	m := maps.Map{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return UserTypeNone, 0, errors.New("decode token error: " + err.Error())
	}

	timestamp := m.GetInt64("timestamp")
	if time.Now().Unix()-timestamp > 600 {
		// 请求超过10分钟认为超时
		return UserTypeNone, 0, errors.New("authenticate timeout")
	}

	t := m.GetString("type")
	if len(userTypes) > 0 && !lists.ContainsString(userTypes, t) {
		return UserTypeNone, 0, errors.New("not supported user type: '" + userType + "'")
	}

	if nodeUserId > 0 {
		return t, nodeUserId, nil
	} else {
		return t, m.GetInt64("userId"), nil
	}
}

// 返回Update成功信息
func RPCUpdateSuccess() (*pb.RPCUpdateSuccess, error) {
	return &pb.RPCUpdateSuccess{}, nil
}

// 返回Delete成功信息
func RPCDeleteSuccess() (*pb.RPCDeleteSuccess, error) {
	return &pb.RPCDeleteSuccess{}, nil
}

// 包装错误
func Wrap(description string, err error) error {
	if err == nil {
		return errors.New(description)
	}
	return errors.New(description + ": " + err.Error())
}
