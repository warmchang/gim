package message

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/protobuf/proto"

	"gim/internal/logic/device"
	"gim/internal/logic/message/domain"
	"gim/internal/logic/message/repo"
	"gim/pkg/grpclib/picker"
	"gim/pkg/md"
	"gim/pkg/mq"
	"gim/pkg/protocol/pb/connectpb"
	pb "gim/pkg/protocol/pb/logicpb"
	"gim/pkg/rpc"
	"gim/pkg/util"
)

const pageSize = 50 // 最大消息同步数量

var App = new(app)

type app struct{}

func (a *app) PushToUserData(ctx context.Context, toUserIDs []uint64, code pb.PushCode, bytes []byte, isPersist bool) (uint64, error) {
	message := pb.Message{
		Code:      code,
		Content:   bytes,
		Seq:       0,
		CreatedAt: 0,
		Status:    0,
	}
	messageID, err := a.SendToUsers(ctx, toUserIDs, &message, isPersist)
	if err != nil {
		slog.Error("PushToUser", "error", err)
		return 0, err
	}
	return messageID, nil
}

func (a *app) PushToUser(ctx context.Context, toUserID []uint64, code pb.PushCode, msg proto.Message, isPersist bool) (uint64, error) {
	bytes, err := proto.Marshal(msg)
	if err != nil {
		slog.Error("PushToUser", "error", err)
		return 0, err
	}
	return a.PushToUserData(ctx, toUserID, code, bytes, isPersist)
}

type userMessageAndDevices struct {
	userMessage *domain.UserMessage
	devices     []*pb.Device
}

// SendToUsers 发送消息给用户
func (a *app) SendToUsers(ctx context.Context, toUserIDs []uint64, message *pb.Message, isPersist bool) (uint64, error) {
	slog.Debug("SendToUser", "request_id", md.GetCtxRequestID(ctx), "to_user_ids", toUserIDs)

	var messageID uint64
	if isPersist {
		message := domain.Message{
			RequestID: md.GetCtxRequestID(ctx),
			Code:      message.Code,
			Content:   message.Content,
		}
		err := repo.MessageRepo.Save(&message)
		if err != nil {
			return 0, err
		}
		messageID = message.ID
	}

	var userMessages []domain.UserMessage
	for _, userID := range toUserIDs {
		var userSeq uint64
		if isPersist {
			var err error
			userSeq, err = repo.SeqRepo.Incr(repo.SeqObjectTypeUser, userID)
			if err != nil {
				return 0, err
			}
		}

		userMessage := domain.UserMessage{
			UserID:    userID,
			Seq:       userSeq,
			MessageID: messageID,
		}
		userMessages = append(userMessages, userMessage)
	}

	err := repo.UserMessageRepo.Save(userMessages)
	if err != nil {
		return 0, err
	}

	devices, err := device.App.ListOnlineByUserID(ctx, toUserIDs)
	if err != nil {
		return 0, err
	}

	userMessageAndDevicesList := make(map[uint64]*userMessageAndDevices, len(userMessages))
	for i := range userMessages {
		userMessageAndDevicesList[userMessages[i].UserID] = &userMessageAndDevices{
			userMessage: &userMessages[i],
			devices:     nil,
		}
	}

	for _, device := range devices {
		value, ok := userMessageAndDevicesList[device.UserId]
		if ok {
			value.devices = append(value.devices, device)
		}
	}

	var deviceAndMessageList = make([]deviceAndMessage, 0, len(devices))
	for _, value := range userMessageAndDevicesList {
		for _, device := range value.devices {
			message.Seq = value.userMessage.Seq
			deviceAndMessageList = append(deviceAndMessageList, deviceAndMessage{
				device:  device,
				message: message,
			})
		}
	}

	err = a.PushToDevices(ctx, deviceAndMessageList)
	return messageID, err
}

type deviceAndMessage struct {
	device  *pb.Device
	message *pb.Message
}

// PushToDevices 将消息发送给设备
func (*app) PushToDevices(ctx context.Context, dms []deviceAndMessage) error {
	connects := make(map[string][]deviceAndMessage)
	for _, dm := range dms {
		connects[dm.device.ConnAddr] = append(connects[dm.device.ConnAddr], dm)
	}

	for addr, dmlist := range connects {
		request := &connectpb.PushToDevicesRequest{
			DeviceMessageList: make([]*connectpb.DeviceMessage, 0, len(dmlist)),
		}
		for _, dm := range dmlist {
			request.DeviceMessageList = append(request.DeviceMessageList, &connectpb.DeviceMessage{
				DeviceId: dm.device.DeviceId,
				Message:  dm.message,
			})
		}

		_, err := rpc.GetConnectIntClient().PushToDevices(picker.ContextWithAddr(ctx, addr), request)
		if err != nil {
			slog.Error("SendToDevice error", "error", err)
			return err
		}
	}

	// todo 其他推送厂商
	return nil
}

// PushAll 全服推送
func (*app) PushAll(ctx context.Context, req *pb.PushAllRequest) error {
	msg := connectpb.PushAllMsg{
		Message: &pb.Message{
			Code:      req.Code,
			Content:   req.Content,
			CreatedAt: util.UnixMilliTime(time.Now()),
		},
	}
	bytes, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}
	return mq.Publish(mq.PushAllTopic, bytes)
}

// Sync 消息同步
func (a *app) Sync(ctx context.Context, userId, seq uint64) (*pb.SyncReply, error) {
	messages, hasMore, err := a.listByUserIdAndSeq(ctx, userId, seq)
	if err != nil {
		return nil, err
	}
	pbMessages := domain.MessagesToPB(messages)

	reply := &pb.SyncReply{Messages: pbMessages, HasMore: hasMore}
	return reply, nil
}

// listByUserIdAndSeq 查询消息
func (a *app) listByUserIdAndSeq(ctx context.Context, userId, seq uint64) ([]domain.UserMessage, bool, error) {
	var err error
	if seq == 0 {
		seq, err = a.getMaxByUserId(ctx, userId)
		if err != nil {
			return nil, false, err
		}
	}
	return repo.UserMessageRepo.ListBySeq(userId, seq, pageSize)
}

// getMaxByUserId 根据用户id获取最大ack
func (*app) getMaxByUserId(ctx context.Context, userId uint64) (uint64, error) {
	acks, err := repo.DeviceACKRepo.Get(userId)
	if err != nil {
		return 0, err
	}

	var max uint64 = 0
	for i := range acks {
		if acks[i] > max {
			max = acks[i]
		}
	}
	return max, nil
}

// MessageAck 收到消息回执
func (*app) MessageAck(ctx context.Context, userId, deviceId, ack uint64) error {
	if ack <= 0 {
		return nil
	}
	return repo.DeviceACKRepo.Set(userId, deviceId, ack)
}
