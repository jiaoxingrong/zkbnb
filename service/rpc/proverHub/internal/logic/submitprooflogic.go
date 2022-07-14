package logic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	cryptoBlock "github.com/zecrey-labs/zecrey-crypto/zecrey-legend/circuit/bn254/block"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/zecrey-labs/zecrey-legend/common/model/blockForProof"
	"github.com/zecrey-labs/zecrey-legend/common/model/proofSender"
	"github.com/zecrey-labs/zecrey-legend/common/util"
	"github.com/zecrey-labs/zecrey-legend/service/rpc/proverHub/internal/svc"
	"github.com/zecrey-labs/zecrey-legend/service/rpc/proverHub/proverHubProto"
)

type SubmitProofLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSubmitProofLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitProofLogic {
	return &SubmitProofLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func packSubmitProofLogic(
	status int64,
	msg string,
	err string,
	result *proverHubProto.ResultSubmitProof,
) (res *proverHubProto.RespSubmitProof) {
	return &proverHubProto.RespSubmitProof{
		Status: status,
		Msg:    msg,
		Err:    err,
		Result: result,
	}
}

func (l *SubmitProofLogic) SubmitProof(in *proverHubProto.ReqSubmitProof) (*proverHubProto.RespSubmitProof, error) {
	var (
		result = &proverHubProto.ResultSubmitProof{}
	)

	// Unmarshal cBlock
	var (
		cBlock *cryptoBlock.Block
	)
	err := json.Unmarshal([]byte(in.BlockInfo), &cBlock)
	if err != nil {
		logx.Errorf("Unmarshal Error: %s", err.Error())
		return packSubmitProofLogic(util.FailStatus, util.FailMsg, err.Error(), result), nil
	}

	// Unmarshal proof
	var (
		proof *util.FormattedProof
	)
	err = json.Unmarshal([]byte(in.Proof), &proof)
	if err != nil {
		logx.Errorf("Unmarshal Error: %s", err.Error())
		return packSubmitProofLogic(util.FailStatus, util.FailMsg, err.Error(), result), nil
	}

	oProof, err := util.UnformatProof(proof)
	if err != nil {
		logx.Errorf("UnformatProof Error: %s", err.Error())
		return packSubmitProofLogic(util.FailStatus, util.FailMsg, err.Error(), result), nil
	}

	vkIndex := 0
	for ; vkIndex < len(VerifyingKeyTxsCount); vkIndex++ {
		if VerifyingKeyTxsCount[vkIndex] == len(cBlock.Txs) {
			break
		}
	}
	// sanity check
	if vkIndex == len(VerifyingKeyTxsCount) {
		logx.Errorf("Can't find correct vk")
		return packSubmitProofLogic(util.FailStatus, util.FailMsg, err.Error(), result), nil
	}
	// VerifyProof
	err = util.VerifyProof(oProof, VerifyingKeys[vkIndex], cBlock)
	if err != nil {
		logx.Errorf("Verify Proof Error: %s", err.Error())
		return packSubmitProofLogic(util.FailStatus, util.FailMsg, err.Error(), result), nil
	}

	// Handle Proof
	// Store Proof and BlockInfo into database and modify the status of UnprovedBlockList

	// check param
	provedBlockModel, err := l.svcCtx.BlockForProofModel.GetUnprovedCryptoBlockByBlockNumber(cBlock.BlockNumber)
	if err != nil {
		logx.Errorf("get provedBlock error, err=%s", err.Error())
		return packSubmitProofLogic(util.FailStatus, util.FailMsg, "get provedBlock error", result), nil
	}

	provedBlock, err := BlockForProofToCryptoBlockInfo(provedBlockModel)
	if err != nil {
		logx.Errorf("marshal crypto block info error, err=%s", err.Error())
		return packSubmitProofLogic(util.FailStatus, util.FailMsg, "marshal crypto block error", result), nil
	}

	// modify UnprovedBlockList
	if provedBlockModel.Status != blockForProof.StatusReceived {
		logx.Errorf("block status error: %d", provedBlockModel.Status)
		return packSubmitProofLogic(util.FailStatus, util.FailMsg, fmt.Sprintf("block status error: %d", provedBlockModel.Status), result), nil
	}

	// check the existence of proof
	_, err = l.svcCtx.ProofSenderModel.GetProofByBlockNumber(provedBlockModel.BlockHeight)
	if err == nil {
		return packSubmitProofLogic(util.FailStatus, util.FailMsg, "proof of current height exists", result), nil
	}

	if common.Bytes2Hex(provedBlock.BlockInfo.NewStateRoot[:]) == common.Bytes2Hex(cBlock.NewStateRoot) &&
		common.Bytes2Hex(provedBlock.BlockInfo.BlockCommitment[:]) == common.Bytes2Hex(cBlock.BlockCommitment) &&
		provedBlock.BlockInfo.CreatedAt == cBlock.CreatedAt {
		var row = &proofSender.ProofSender{
			ProofInfo:   in.Proof,
			BlockNumber: cBlock.BlockNumber,
			Status:      proofSender.NotSent,
		}
		err = l.svcCtx.ProofSenderModel.CreateProof(row)
		if err != nil {
			// rollback the status, it is fine if it fails updating the status for we will check the timeout
			_ = l.svcCtx.BlockForProofModel.UpdateUnprovedCryptoBlockStatus(provedBlockModel, blockForProof.StatusPublished)
			logx.Error("CreateProof error")
			return packSubmitProofLogic(util.FailStatus, util.FailMsg, err.Error(), result), nil
		}
		logx.Infof("Block %d CreateProof Successfully!", cBlock.BlockNumber)
	} else {
		logx.Error("data inconsistency error")
		return packSubmitProofLogic(util.FailStatus, util.FailMsg, "data inconsistency", result), nil
	}

	return packSubmitProofLogic(util.SuccessStatus, util.SuccessMsg, util.NilErrorString, result), nil
}
