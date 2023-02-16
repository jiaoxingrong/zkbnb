package transaction

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/bnb-chain/zkbnb/dao/tx"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/logic/utils"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/svc"
	"github.com/bnb-chain/zkbnb/service/apiserver/internal/types"
	types2 "github.com/bnb-chain/zkbnb/types"
)

const (
	queryByAccountIndex = "account_index"
	queryByAccountName  = "account_name"
	queryByAccountPk    = "account_pk"
)

type GetAccountTxsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAccountTxsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAccountTxsLogic {
	return &GetAccountTxsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAccountTxsLogic) GetAccountTxs(req *types.ReqGetAccountTxs) (resp *types.Txs, err error) {
	accountIndex, err := l.fetchAccountIndexFromReq(req)
	if err != nil {
		if err == types2.DbErrNotFound {
			return resp, nil
		}
		return nil, types2.AppErrInternal
	}

	var options []tx.GetTxOptionFunc
	if len(req.Types) > 0 {
		options = append(options, tx.GetTxWithTypes(req.Types))
	}

	total, err := l.svcCtx.TxModel.GetTxsCountByAccountIndex(accountIndex, options...)
	if err != nil {
		return nil, types2.AppErrInternal
	}

	if total == 0 || total <= int64(req.Offset) {
		return resp, nil
	}

	txs, err := l.svcCtx.TxModel.GetTxsByAccountIndex(accountIndex, int64(req.Limit), int64(req.Offset), options...)
	if err != nil {
		return nil, types2.AppErrInternal
	}

	resp = l.convertTxsList(uint32(total), txs)
	return resp, nil
}

func (l *GetAccountTxsLogic) fetchAccountIndexFromReq(req *types.ReqGetAccountTxs) (int64, error) {
	switch req.By {
	case queryByAccountIndex:
		accountIndex, err := strconv.ParseInt(req.Value, 10, 64)
		if err != nil || accountIndex < 0 {
			return accountIndex, types2.AppErrInvalidAccountIndex
		}
		return accountIndex, err
	case queryByAccountName:
		accountIndex, err := l.svcCtx.MemCache.GetAccountIndexByName(req.Value)
		return accountIndex, err
	case queryByAccountPk:
		accountIndex, err := l.svcCtx.MemCache.GetAccountIndexByPk(req.Value)
		return accountIndex, err
	}
	return 0, types2.AppErrInvalidParam.RefineError("param by should be account_index|account_name|account_pk")
}

func (l *GetAccountTxsLogic) convertTxsList(totalCount uint32, txList []*tx.Tx) *types.Txs {

	resp := &types.Txs{
		Txs: make([]*types.Tx, 0, totalCount),
	}
	for _, dbTx := range txList {
		tx := utils.ConvertTx(dbTx)
		tx.AccountName, _ = l.svcCtx.MemCache.GetAccountNameByIndex(tx.AccountIndex)
		tx.AssetName, _ = l.svcCtx.MemCache.GetAssetNameById(tx.AssetId)
		if tx.ToAccountIndex >= 0 {
			tx.ToAccountName, _ = l.svcCtx.MemCache.GetAccountNameByIndex(tx.ToAccountIndex)
		}
		resp.Txs = append(resp.Txs, tx)
	}
	return resp
}
