package v1

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/proto/gen/api/v1/apiv1connect"
)

// ConnectServiceHandler wraps APIV1Service to implement Connect handler interfaces.
// It adapts the existing gRPC service implementations to work with Connect's
// request/response wrapper types.
//
// This wrapper pattern allows us to:
// - Reuse existing gRPC service implementations
// - Support both native gRPC and Connect protocols
// - Maintain a single source of truth for business logic.
type ConnectServiceHandler struct {
	*APIV1Service
}

// NewConnectServiceHandler creates a new Connect service handler.
func NewConnectServiceHandler(svc *APIV1Service) *ConnectServiceHandler {
	return &ConnectServiceHandler{APIV1Service: svc}
}

// RegisterConnectHandlers registers all Connect service handlers on the given mux.
func (s *ConnectServiceHandler) RegisterConnectHandlers(mux *http.ServeMux, opts ...connect.HandlerOption) {
	// Register all service handlers
	handlers := []struct {
		handler http.Handler
		path    string
	}{
		wrap(apiv1connect.NewInstanceServiceHandler(s, opts...)),
		wrap(apiv1connect.NewAuthServiceHandler(s, opts...)),
		wrap(apiv1connect.NewUserServiceHandler(s, opts...)),
		wrap(apiv1connect.NewMemoServiceHandler(s, opts...)),
		wrap(apiv1connect.NewAttachmentServiceHandler(s, opts...)),
		wrap(apiv1connect.NewShortcutServiceHandler(s, opts...)),
		wrap(apiv1connect.NewActivityServiceHandler(s, opts...)),
		wrap(apiv1connect.NewIdentityProviderServiceHandler(s, opts...)),
	}

	if s.AIService != nil {
		handlers = append(handlers, wrap(apiv1connect.NewAIServiceHandler(s, opts...)))
	}

	// Register Schedule service handlers
	handlers = append(handlers, wrap(apiv1connect.NewScheduleServiceHandler(s, opts...)))

	for _, h := range handlers {
		mux.Handle(h.path, h.handler)
	}
}

// wrap converts (path, handler) return value to a struct for cleaner iteration.
func wrap(path string, handler http.Handler) struct {
	handler http.Handler
	path    string
} {
	return struct {
		handler http.Handler
		path    string
	}{
		handler: handler,
		path:    path,
	}
}

// convertGRPCError converts gRPC status errors to Connect errors.
// This preserves the error code semantics between the two protocols.
func convertGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok {
		return connect.NewError(grpcCodeToConnectCode(st.Code()), err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

// grpcCodeToConnectCode converts gRPC status codes to Connect error codes.
// gRPC and Connect use the same error code semantics, so this is a direct cast.
// See: https://connectrpc.com/docs/protocol/#error-codes
func grpcCodeToConnectCode(code codes.Code) connect.Code {
	return connect.Code(code)
}

// AIService wrappers for Connect

func (s *ConnectServiceHandler) SuggestTags(ctx context.Context, req *connect.Request[v1pb.SuggestTagsRequest]) (*connect.Response[v1pb.SuggestTagsResponse], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.SuggestTags(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) Format(ctx context.Context, req *connect.Request[v1pb.FormatRequest]) (*connect.Response[v1pb.FormatResponse], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.Format(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) Summary(ctx context.Context, req *connect.Request[v1pb.SummaryRequest]) (*connect.Response[v1pb.SummaryResponse], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.Summary(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) SemanticSearch(ctx context.Context, req *connect.Request[v1pb.SemanticSearchRequest]) (*connect.Response[v1pb.SemanticSearchResponse], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.SemanticSearch(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetRelatedMemos(ctx context.Context, req *connect.Request[v1pb.GetRelatedMemosRequest]) (*connect.Response[v1pb.GetRelatedMemosResponse], error) {
	// Use AI-powered implementation when available, otherwise fallback to keyword-based
	if s.AIService != nil && s.AIService.IsEnabled() {
		resp, err := s.AIService.GetRelatedMemos(ctx, req.Msg)
		if err != nil {
			return nil, convertGRPCError(err)
		}
		return connect.NewResponse(resp), nil
	}
	// Fallback to MemoService implementation (keyword/tag-based)
	resp, err := s.APIV1Service.GetRelatedMemos(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) Chat(ctx context.Context, req *connect.Request[v1pb.ChatRequest], stream *connect.ServerStream[v1pb.ChatResponse]) error {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}

	// Log entry for debugging
	slog.Debug("ConnectServiceHandler: Chat called",
		"message", truncateStringForLog(req.Msg.Message, 50),
		"agent_type", req.Msg.AgentType.String(),
		"agent_type_value", int(req.Msg.AgentType),
		"is_default", req.Msg.AgentType == v1pb.AgentType_AGENT_TYPE_DEFAULT,
	)

	// Delegate to AIService.Chat which has the full agent routing logic
	return s.AIService.Chat(req.Msg, &connectStreamAdapter{
		stream: stream,
		ctx:    ctx,
	})
}

// connectStreamAdapter wraps Connect ServerStream to implement AIService_ChatServer.
type connectStreamAdapter struct {
	stream *connect.ServerStream[v1pb.ChatResponse]
	ctx    context.Context
}

func (a *connectStreamAdapter) Send(resp *v1pb.ChatResponse) error {
	return a.stream.Send(resp)
}

func (a *connectStreamAdapter) Context() context.Context {
	return a.ctx
}

func (a *connectStreamAdapter) SendMsg(m any) error {
	if resp, ok := m.(*v1pb.ChatResponse); ok {
		return a.Send(resp)
	}
	return fmt.Errorf("invalid message type: %T", m)
}

func (a *connectStreamAdapter) RecvMsg(m any) error {
	return fmt.Errorf("RecvMsg not supported for server streaming")
}

func (a *connectStreamAdapter) SetHeader(md metadata.MD) error {
	return nil
}

func (a *connectStreamAdapter) SendHeader(md metadata.MD) error {
	return nil
}

func (a *connectStreamAdapter) SetTrailer(md metadata.MD) {
}

// truncateStringForLog truncates a string for logging.
func truncateStringForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// ScheduleService wrappers for Connect

func (s *ConnectServiceHandler) CreateSchedule(ctx context.Context, req *connect.Request[v1pb.CreateScheduleRequest]) (*connect.Response[v1pb.Schedule], error) {
	resp, err := s.ScheduleService.CreateSchedule(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListSchedules(ctx context.Context, req *connect.Request[v1pb.ListSchedulesRequest]) (*connect.Response[v1pb.ListSchedulesResponse], error) {
	resp, err := s.ScheduleService.ListSchedules(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetSchedule(ctx context.Context, req *connect.Request[v1pb.GetScheduleRequest]) (*connect.Response[v1pb.Schedule], error) {
	resp, err := s.ScheduleService.GetSchedule(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpdateSchedule(ctx context.Context, req *connect.Request[v1pb.UpdateScheduleRequest]) (*connect.Response[v1pb.Schedule], error) {
	resp, err := s.ScheduleService.UpdateSchedule(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeleteSchedule(ctx context.Context, req *connect.Request[v1pb.DeleteScheduleRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.ScheduleService.DeleteSchedule(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) CheckConflict(ctx context.Context, req *connect.Request[v1pb.CheckConflictRequest]) (*connect.Response[v1pb.CheckConflictResponse], error) {
	resp, err := s.ScheduleService.CheckConflict(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ParseAndCreateSchedule(ctx context.Context, req *connect.Request[v1pb.ParseAndCreateScheduleRequest]) (*connect.Response[v1pb.ParseAndCreateScheduleResponse], error) {
	resp, err := s.ScheduleService.ParseAndCreateSchedule(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) PrecheckSchedule(ctx context.Context, req *connect.Request[v1pb.PrecheckScheduleRequest]) (*connect.Response[v1pb.PrecheckScheduleResponse], error) {
	resp, err := s.ScheduleService.PrecheckSchedule(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) BatchParseSchedule(ctx context.Context, req *connect.Request[v1pb.BatchParseScheduleRequest]) (*connect.Response[v1pb.BatchParseScheduleResponse], error) {
	resp, err := s.ScheduleService.BatchParseSchedule(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) BatchCreateSchedules(ctx context.Context, req *connect.Request[v1pb.BatchCreateSchedulesRequest]) (*connect.Response[v1pb.BatchCreateSchedulesResponse], error) {
	resp, err := s.ScheduleService.BatchCreateSchedules(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// GetParrotSelfCognition returns the metacognitive information of a parrot agent.
func (s *ConnectServiceHandler) GetParrotSelfCognition(ctx context.Context, req *connect.Request[v1pb.GetParrotSelfCognitionRequest]) (*connect.Response[v1pb.GetParrotSelfCognitionResponse], error) {
	agentType := req.Msg.GetAgentType()
	selfCognition := getParrotSelfCognition(agentType)

	return connect.NewResponse(&v1pb.GetParrotSelfCognitionResponse{
		SelfCognition: selfCognition,
	}), nil
}

// ListParrots returns all available parrot agents with their metacognitive information.
func (s *ConnectServiceHandler) ListParrots(ctx context.Context, req *connect.Request[v1pb.ListParrotsRequest]) (*connect.Response[v1pb.ListParrotsResponse], error) {
	// Return all available parrot types
	agentTypes := []v1pb.AgentType{
		v1pb.AgentType_AGENT_TYPE_DEFAULT,
		v1pb.AgentType_AGENT_TYPE_MEMO,
		v1pb.AgentType_AGENT_TYPE_SCHEDULE,
		v1pb.AgentType_AGENT_TYPE_GENERAL,
		v1pb.AgentType_AGENT_TYPE_IDEATION,
	}

	parrots := make([]*v1pb.ParrotInfo, 0, len(agentTypes))
	for _, agentType := range agentTypes {
		parrots = append(parrots, &v1pb.ParrotInfo{
			AgentType:     agentType,
			Name:          getParrotNameByAgentType(agentType),
			SelfCognition: getParrotSelfCognition(agentType),
		})
	}

	return connect.NewResponse(&v1pb.ListParrotsResponse{
		Parrots: parrots,
	}), nil
}

// Helper function to get parrot self-cognition by agent type.
// NOTE: FavoriteTools lists PRIMARY functional tools only (excludes utility tools like report_inability).
// Capabilities should match self_description.capabilities in config/parrots/*.yaml.
func getParrotSelfCognition(agentType v1pb.AgentType) *v1pb.ParrotSelfCognition {
	switch agentType {
	case v1pb.AgentType_AGENT_TYPE_MEMO:
		return &v1pb.ParrotSelfCognition{
			Name:             "memo",
			Emoji:            "📝",
			Title:            "灰灰 - 笔记搜索专家",
			Personality:      []string{"精准", "高效"},
			Capabilities:     []string{"搜索笔记内容", "按时间浏览笔记", "查找相关内容"},
			WorkingStyle:     "先理解问题，检索相关笔记，然后综合分析给出答案",
			FavoriteTools:    []string{"memo_search"},
			SelfIntroduction: "我是灰灰，您的笔记搜索专家。我擅长通过语义搜索精准找到您笔记中的相关信息。",
			FunFact:          "我是一只非洲灰鹦鹉，以记忆力和智慧著称",
		}
	case v1pb.AgentType_AGENT_TYPE_SCHEDULE:
		return &v1pb.ParrotSelfCognition{
			Name:             "schedule",
			Emoji:            "📅",
			Title:            "时巧 - 日程管理专家",
			Personality:      []string{"严谨", "周全"},
			Capabilities:     []string{"创建日程", "查询日程", "更新日程", "查找空闲时间"},
			WorkingStyle:     "分析时间需求，查询现有日程，帮助安排和提醒",
			FavoriteTools:    []string{"schedule_add", "schedule_query", "schedule_update", "schedule_delete", "find_free_time"},
			SelfIntroduction: "我是时巧，您的日程管理专家。我帮您管理时间、安排日程、智能处理时间冲突。",
			FunFact:          "我是一只鸡尾鹦鹉，以守时和灵巧著称",
		}
	case v1pb.AgentType_AGENT_TYPE_GENERAL:
		return &v1pb.ParrotSelfCognition{
			Name:             "general",
			Emoji:            "🤖",
			Title:            "通才 - 通用智能代理",
			Personality:      []string{"全能", "灵活", "善于应变"},
			Capabilities:     []string{"文本总结", "翻译转换", "改写润色", "问答对话", "分析推理"},
			WorkingStyle:     "灵活应对各类通用任务，提供简洁有效的帮助",
			FavoriteTools:    []string{}, // General uses no tools, pure LLM
			SelfIntroduction: "我是通才，您的通用智能代理。我能帮您处理总结、翻译、改写、问答等各类纯文本任务。",
			FunFact:          "我是一只灰鹦鹉，以聪明和适应力强著称",
		}
	case v1pb.AgentType_AGENT_TYPE_IDEATION:
		return &v1pb.ParrotSelfCognition{
			Name:             "ideation",
			Emoji:            "💡",
			Title:            "灵光 - 创意生成专家",
			Personality:      []string{"创意", "灵感丰富", "思维活跃"},
			Capabilities:     []string{"头脑风暴", "写作辅助", "创意生成", "灵感激发", "产品创意", "营销文案", "故事构思"},
			WorkingStyle:     "激发创意思维，提供多角度方案，鼓励创新",
			FavoriteTools:    []string{"memo_write"},
			SelfIntroduction: "我是灵光，您的创意生成专家。我擅长头脑风暴、创意写作和灵感激发。",
			FunFact:          "我是一只紫罗兰鹦鹉，以创意和灵感著称",
		}
	default:
		return &v1pb.ParrotSelfCognition{
			Name:             "default",
			Emoji:            "🤖",
			Title:            "默认助手",
			Personality:      []string{"通用", "友好", "乐于助人"},
			Capabilities:     []string{"通用对话", "问答"},
			WorkingStyle:     "理解问题，提供帮助",
			FavoriteTools:    []string{},
			SelfIntroduction: "我是您的 AI 助手，随时准备帮助您。",
			FunFact:          "我是默认助手，什么都会一点",
		}
	}
}

// Helper function to get parrot name (identifier) by agent type.
// Returns English identifier for programmatic use; display name is in SelfCognition.Title.
func getParrotNameByAgentType(agentType v1pb.AgentType) string {
	switch agentType {
	case v1pb.AgentType_AGENT_TYPE_MEMO:
		return "memo"
	case v1pb.AgentType_AGENT_TYPE_SCHEDULE:
		return "schedule"
	case v1pb.AgentType_AGENT_TYPE_GENERAL:
		return "general"
	case v1pb.AgentType_AGENT_TYPE_IDEATION:
		return "ideation"
	default:
		return "auto"
	}
}

// AIConversation Connect wrappers

func (s *ConnectServiceHandler) ListAIConversations(ctx context.Context, req *connect.Request[v1pb.ListAIConversationsRequest]) (*connect.Response[v1pb.ListAIConversationsResponse], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.ListAIConversations(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetAIConversation(ctx context.Context, req *connect.Request[v1pb.GetAIConversationRequest]) (*connect.Response[v1pb.AIConversation], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.GetAIConversation(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) CreateAIConversation(ctx context.Context, req *connect.Request[v1pb.CreateAIConversationRequest]) (*connect.Response[v1pb.AIConversation], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.CreateAIConversation(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpdateAIConversation(ctx context.Context, req *connect.Request[v1pb.UpdateAIConversationRequest]) (*connect.Response[v1pb.AIConversation], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.UpdateAIConversation(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GenerateConversationTitle(ctx context.Context, req *connect.Request[v1pb.GenerateConversationTitleRequest]) (*connect.Response[v1pb.GenerateConversationTitleResponse], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.GenerateConversationTitle(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeleteAIConversation(ctx context.Context, req *connect.Request[v1pb.DeleteAIConversationRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.DeleteAIConversation(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) AddContextSeparator(ctx context.Context, req *connect.Request[v1pb.AddContextSeparatorRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.AddContextSeparator(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ClearConversationMessages(ctx context.Context, req *connect.Request[v1pb.ClearConversationMessagesRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.ClearConversationMessages(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// StopChat cancels an ongoing chat stream and terminates the associated session.
// StopChat 取消正在进行的聊天流并终止相关会话。
func (s *ConnectServiceHandler) StopChat(ctx context.Context, req *connect.Request[v1pb.StopChatRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.StopChat(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DetectDuplicates(ctx context.Context, req *connect.Request[v1pb.DetectDuplicatesRequest]) (*connect.Response[v1pb.DetectDuplicatesResponse], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.DetectDuplicates(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) MergeMemos(ctx context.Context, req *connect.Request[v1pb.MergeMemosRequest]) (*connect.Response[v1pb.MergeMemosResponse], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.MergeMemos(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) LinkMemos(ctx context.Context, req *connect.Request[v1pb.LinkMemosRequest]) (*connect.Response[v1pb.LinkMemosResponse], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.LinkMemos(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetKnowledgeGraph(ctx context.Context, req *connect.Request[v1pb.GetKnowledgeGraphRequest]) (*connect.Response[v1pb.GetKnowledgeGraphResponse], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.GetKnowledgeGraph(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetDueReviews(ctx context.Context, req *connect.Request[v1pb.GetDueReviewsRequest]) (*connect.Response[v1pb.GetDueReviewsResponse], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.GetDueReviews(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) RecordReview(ctx context.Context, req *connect.Request[v1pb.RecordReviewRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.RecordReview(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) RecordRouterFeedback(ctx context.Context, req *connect.Request[v1pb.RecordRouterFeedbackRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	_, err := s.AIService.RecordRouterFeedback(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *ConnectServiceHandler) GetReviewStats(ctx context.Context, req *connect.Request[v1pb.GetReviewStatsRequest]) (*connect.Response[v1pb.GetReviewStatsResponse], error) {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.GetReviewStats(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// MemoService wrappers for Connect

func (s *ConnectServiceHandler) SearchWithHighlight(ctx context.Context, req *connect.Request[v1pb.SearchWithHighlightRequest]) (*connect.Response[v1pb.SearchWithHighlightResponse], error) {
	resp, err := s.APIV1Service.SearchWithHighlight(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// AIBlock (Unified Block Model) wrappers for Connect

func (s *ConnectServiceHandler) ListBlocks(ctx context.Context, req *connect.Request[v1pb.ListBlocksRequest]) (*connect.Response[v1pb.ListBlocksResponse], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.ListBlocks(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetBlock(ctx context.Context, req *connect.Request[v1pb.GetBlockRequest]) (*connect.Response[v1pb.Block], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.GetBlock(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) CreateBlock(ctx context.Context, req *connect.Request[v1pb.CreateBlockRequest]) (*connect.Response[v1pb.Block], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.CreateBlock(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpdateBlock(ctx context.Context, req *connect.Request[v1pb.UpdateBlockRequest]) (*connect.Response[v1pb.Block], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.UpdateBlock(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeleteBlock(ctx context.Context, req *connect.Request[v1pb.DeleteBlockRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.DeleteBlock(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) AppendUserInput(ctx context.Context, req *connect.Request[v1pb.AppendUserInputRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.AppendUserInput(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) AppendEvent(ctx context.Context, req *connect.Request[v1pb.AppendEventRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.AppendEvent(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// ========== Tree Branching Methods (tree-conversation-branching) ==========

func (s *ConnectServiceHandler) ForkBlock(ctx context.Context, req *connect.Request[v1pb.ForkBlockRequest]) (*connect.Response[v1pb.Block], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.ForkBlock(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListBlockBranches(ctx context.Context, req *connect.Request[v1pb.ListBlockBranchesRequest]) (*connect.Response[v1pb.ListBlockBranchesResponse], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.ListBlockBranches(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) SwitchBranch(ctx context.Context, req *connect.Request[v1pb.SwitchBranchRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.SwitchBranch(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeleteBranch(ctx context.Context, req *connect.Request[v1pb.DeleteBranchRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.AIService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	resp, err := s.AIService.DeleteBranch(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}
