package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/scenic-guide/internal/pkg"
)

// SetChatSessionService 注入会话持久化服务（后期注入，避免循环依赖）
func (s *RAGService) SetChatSessionService(svc *ChatSessionService) {
	s.chatSessionService = svc
}

func (s *RAGService) QueryWithRAGInSession(ctx context.Context, sessionID, query, lang string) (string, error) {
	answer, _, err := s.QueryWithRAGTraceInSession(ctx, sessionID, query, lang)
	return answer, err
}

func (s *RAGService) QueryWithRAGTraceInSession(ctx context.Context, sessionID, query, lang string) (string, RAGTrace, error) {
	sessionID = normalizeSessionID(sessionID)
	if sessionID == "" {
		return s.QueryWithRAGTrace(ctx, query, lang)
	}
	// 缓存未命中时尝试从 DB 加载历史
	s.cacheMutex.RLock()
	_, hasHistory := s.sessionHistory[sessionID]
	s.cacheMutex.RUnlock()
	if !hasHistory {
		s.LoadSessionHistoryFromDB(sessionID)
	}
	rewritten := s.RewriteFollowUpQuery(sessionID, query)
	sessionContext := s.buildSessionContextText(sessionID, query)
	answer, trace, err := s.queryWithRAGTraceInternal(ctx, rewritten, query, sessionContext, lang)
	if rewritten != query {
		trace.RewrittenQuery = rewritten
	}
	if err != nil {
		return "", trace, err
	}
	s.appendSessionTurn(sessionID, query, answer)
	return answer, trace, nil
}

func (s *RAGService) RewriteFollowUpQuery(sessionID, query string) string {
	sessionID = normalizeSessionID(sessionID)
	query = strings.TrimSpace(query)
	if query == "" || sessionID == "" {
		return query
	}
	if !isFollowUpQuery(query) {
		return query
	}

	s.cacheMutex.RLock()
	history := append([]sessionTurn(nil), s.sessionHistory[sessionID]...)
	s.cacheMutex.RUnlock()
	if len(history) == 0 {
		return query
	}

	return s.buildFollowUpRewrite(query, history)
}

func (s *RAGService) appendTurnLocked(sessionID, query, answer string) {
	now := time.Now()
	ctx := s.inferConversationContext(query, answer)
	turns := append(s.sessionHistory[sessionID], sessionTurn{
		Query:    strings.TrimSpace(query),
		Answer:   strings.TrimSpace(answer),
		Topic:    ctx.Topic,
		Intent:   ctx.Intent,
		Boundary: ctx.Boundary,
		Updated:  now,
	})
	if len(turns) > MaxCachedTurns {
		turns = turns[len(turns)-MaxCachedTurns:]
	}
	s.sessionHistory[sessionID] = turns
	s.cleanupSessionHistoryLocked(now)
}

// appendSessionTurn 仅更新内存中的会话历史缓存(用于追问改写、上下文构建),
// 不再持久化到数据库。
//
// 设计说明:此前本函数会在内部异步调用 AddMessages 写库,而 Handler 层
// (ai_handler / ai_proxy_handler)又会调用 AppendSessionTurnWithUser 再次写库,
// 导致同一轮对话被持久化两次(消息历史冗余、统计翻倍)。现把写库职责统一收口到
// AppendSessionTurnWithUser(它携带真实 userID,可正确归属消息),本函数只负责内存缓存。
// 流式路径(QueryWithRAGStreaming)本就不调用此函数,仅由 Handler 写一次;非流式路径
// (QueryWithRAGTraceInSession)调用此函数更新内存后,同样由 Handler 写一次。
func (s *RAGService) appendSessionTurn(sessionID, query, answer string) {
	sessionID = normalizeSessionID(sessionID)
	if sessionID == "" {
		return
	}
	s.cacheMutex.Lock()
	s.appendTurnLocked(sessionID, query, answer)
	s.cacheMutex.Unlock()
}

// AppendSessionTurnWithUser 带用户ID的会话写入(供 Handler 层调用)。
// 同时更新内存缓存与异步持久化到数据库,是会话消息的唯一写库入口。
func (s *RAGService) AppendSessionTurnWithUser(sessionID string, userID uint, query, answer string) {
	sessionID = normalizeSessionID(sessionID)
	if sessionID == "" {
		return
	}
	if s.chatSessionService != nil {
		pkg.SafeGo("AppendSessionTurnWithUser", func() {
			if err := s.chatSessionService.AddMessages(sessionID, userID, query, answer, "", 0); err != nil {
				slog.Warn("会话持久化失败", "session_id", sessionID, "user_id", userID, "error", err)
			}
		})
	}
}

// LoadSessionHistoryFromDB 从数据库加载会话历史到内存缓存
func (s *RAGService) LoadSessionHistoryFromDB(sessionID string) {
	sessionID = normalizeSessionID(sessionID)
	if sessionID == "" || s.chatSessionService == nil {
		return
	}

	// 检查内存缓存是否已有数据
	s.cacheMutex.RLock()
	if turns, ok := s.sessionHistory[sessionID]; ok && len(turns) > 0 {
		s.cacheMutex.RUnlock()
		return
	}
	s.cacheMutex.RUnlock()

	// 从 DB 加载最近消息
	msgs, err := s.chatSessionService.GetRecentMessages(sessionID, MaxCachedTurns*2)
	if err != nil || len(msgs) == 0 {
		return
	}

	// 将消息对转换为 sessionTurn
	var turns []sessionTurn
	for i := 0; i+1 < len(msgs); i += 2 {
		if msgs[i].Role == "user" && msgs[i+1].Role == "assistant" {
			ctx := s.inferConversationContext(msgs[i].Content, msgs[i+1].Content)
			turns = append(turns, sessionTurn{
				Query:    msgs[i].Content,
				Answer:   msgs[i+1].Content,
				Topic:    ctx.Topic,
				Intent:   ctx.Intent,
				Boundary: ctx.Boundary,
				Updated:  msgs[i+1].CreatedAt,
			})
		}
	}
	if len(turns) > MaxCachedTurns {
		turns = turns[len(turns)-MaxCachedTurns:]
	}

	s.cacheMutex.Lock()
	// 再次检查（避免并发加载覆盖）
	if existing, ok := s.sessionHistory[sessionID]; !ok || len(existing) == 0 {
		s.sessionHistory[sessionID] = turns
	}
	s.cacheMutex.Unlock()
}

func normalizeSessionID(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if len(sessionID) > MaxSessionIDLength {
		return sessionID[:MaxSessionIDLength]
	}
	return sessionID
}

func (s *RAGService) cleanupSessionHistoryLocked(now time.Time) {
	type candidate struct {
		id      string
		updated time.Time
	}
	candidates := make([]candidate, 0, len(s.sessionHistory))
	for id, turns := range s.sessionHistory {
		if len(turns) == 0 {
			delete(s.sessionHistory, id)
			continue
		}
		updated := turns[len(turns)-1].Updated
		if now.Sub(updated) > SessionHistoryTTL {
			delete(s.sessionHistory, id)
			continue
		}
		candidates = append(candidates, candidate{id: id, updated: updated})
	}
	if len(s.sessionHistory) <= MaxSessionHistorySize {
		return
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].updated.Equal(candidates[j].updated) {
			return candidates[i].id < candidates[j].id
		}
		return candidates[i].updated.Before(candidates[j].updated)
	})
	for _, item := range candidates {
		if len(s.sessionHistory) <= MaxSessionHistorySize {
			return
		}
		delete(s.sessionHistory, item.id)
	}
}

func isFollowUpQuery(query string) bool {
	return containsAny(query, []string{
		"它", "那里", "刚才", "那个", "这个", "呢", "还有", "别的", "多少", "几点", "门票", "怎么去", "哪里", "在哪", "多高", "要多久",
		"半天够吗", "先看什么", "下雨", "雨天", "带孩子", "小朋友", "老人", "今天", "现在", "开不开", "人多", "排队", "无人机", "宠物",
	})
}

func (s *RAGService) buildSessionContextText(sessionID, query string) string {
	s.cacheMutex.RLock()
	history := append([]sessionTurn(nil), s.sessionHistory[sessionID]...)
	s.cacheMutex.RUnlock()
	if len(history) == 0 {
		return ""
	}
	currentIntent := detectQuestionIntent(query)
	last := latestContextTurn(history)
	parts := make([]string, 0, 3)
	if last.Topic != "" {
		parts = append(parts, "上一轮主题："+last.Topic)
	}
	if currentIntent != "" {
		parts = append(parts, "当前意图："+currentIntent)
	} else if last.Intent != "" {
		parts = append(parts, "上一轮意图："+last.Intent)
	}
	if isBoundaryIntent(query) || last.Boundary {
		parts = append(parts, "边界状态：涉及实时信息或现场规则时不能直接承诺")
	}
	if len(parts) == 0 {
		return ""
	}
	return "- " + strings.Join(parts, "\n- ")
}

func (s *RAGService) inferConversationContext(query, answer string) conversationContext {
	text := query + "\n" + answer
	intent := detectQuestionIntent(query)
	if intent == "" {
		intent = detectQuestionIntent(answer)
	}
	return conversationContext{
		Topic:    s.detectTopicEntity(text),
		Intent:   intent,
		Boundary: isBoundaryIntent(query) || isBoundaryIntent(answer),
	}
}

func detectQuestionIntent(query string) string {
	switch {
	case containsAny(query, []string{"今天", "现在", "开不开", "开放", "几点", "门票", "票价", "演出", "场次", "人多", "排队", "无人机", "宠物", "公告", "现场"}):
		return "实时信息边界"
	case containsAny(query, []string{"下雨", "雨天", "天气", "高温"}):
		return "天气路线"
	case containsAny(query, []string{"半天", "路线", "怎么走", "先看", "够吗"}):
		return "路线规划"
	case containsAny(query, []string{"带孩子", "小朋友", "亲子"}):
		return "亲子路线"
	case containsAny(query, []string{"老人", "长辈", "腿脚"}):
		return "老人路线"
	case containsAny(query, []string{"多高", "在哪", "哪里", "怎么去", "适合谁", "要多久", "讲什么", "是什么"}):
		return "属性追问"
	case containsAny(query, []string{"还有", "别的"}):
		return "补充推荐"
	default:
		return ""
	}
}

// detectTopicEntity 从文本中检测话题实体（配置化，支持任意景区）
func (s *RAGService) detectTopicEntity(text string) string {
	for _, topic := range s.getTopicEntities() {
		if strings.Contains(text, topic) {
			return topic
		}
	}
	switch {
	case containsAny(text, []string{"半天", "主线", "中轴线", "先看", "路线"}):
		return "路线"
	case containsAny(text, []string{"下雨", "雨天", "天气", "高温"}):
		return "天气路线"
	case containsAny(text, []string{"带孩子", "小朋友", "亲子"}):
		return "亲子路线"
	case containsAny(text, []string{"老人", "长辈", "腿脚"}):
		return "老人路线"
	case containsAny(text, []string{"导览服务", "服务中心", "洗手间", "休息点"}):
		return "导览服务"
	case isBoundaryIntent(text):
		return "实时信息边界"
	default:
		return ""
	}
}

func isBoundaryIntent(query string) bool {
	return containsAny(query, []string{"今天", "现在", "现场", "开不开", "开放", "几点", "门票", "票价", "演出", "场次", "人多", "排队", "无人机", "宠物", "公告", "实时", "不能替代", "不能编造", "酒店空房", "还有多少间", "房态", "剩余房间", "客房库存"})
}

// buildFollowUpRewrite 构建追问改写查询（配置化，支持任意景区）
func (s *RAGService) buildFollowUpRewrite(query string, history []sessionTurn) string {
	last := latestContextTurn(history)
	topic := last.Topic
	if topic == "" {
		topic = s.detectTopicEntity(last.Query + "\n" + last.Answer)
	}
	intent := detectQuestionIntent(query)
	if intent == "" {
		intent = last.Intent
	}

	terms := []string{topic, query}

	// 实时信息和属性追问使用通用规则
	if intent == "实时信息边界" {
		terms = append(terms, boundaryRewriteTerms(query)...)
	} else if intent == "属性追问" {
		terms = append(terms, attributeRewriteTerms(query)...)
	} else if s.profile != nil && s.profile.Prompts.FollowUpRewrite != nil {
		// 从 profile 配置读取意图→关键词映射
		if termsStr, ok := s.profile.Prompts.FollowUpRewrite[intent]; ok {
			terms = append(terms, strings.Fields(termsStr)...)
		}
		// 天气路线：如果上一轮是路线规划，补充路线关键词
		if intent == "天气路线" && (topic == "路线" || last.Intent == "路线规划") {
			terms = append(terms, "路线")
		}
	}

	terms = compactKeywords(terms)
	if len(terms) == 0 {
		return query
	}
	return strings.Join(terms, " ")
}

func latestContextTurn(history []sessionTurn) sessionTurn {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Topic != "" || history[i].Intent != "" || strings.TrimSpace(history[i].Query) != "" {
			return history[i]
		}
	}
	return sessionTurn{}
}

func boundaryRewriteTerms(query string) []string {
	terms := []string{"官方最新公告", "现场公示", "实时信息", "不能编造"}
	if containsAny(query, []string{"人多", "排队"}) {
		terms = append(terms, "实时客流", "排队时间", "无法确认")
	}
	if containsAny(query, []string{"无人机", "宠物"}) {
		terms = append(terms, "无人机", "宠物", "现场规定", "正式规定")
	}
	if containsAny(query, []string{"门票", "票价", "几点", "开不开", "开放"}) {
		terms = append(terms, "门票", "开放时间", "以官方为准")
	}
	return terms
}

func attributeRewriteTerms(query string) []string {
	terms := make([]string, 0, 4)
	if containsAny(query, []string{"多高", "高度"}) {
		terms = append(terms, "高度", "通高")
	}
	if containsAny(query, []string{"在哪", "哪里", "怎么去"}) {
		terms = append(terms, "位置", "怎么去", "路线")
	}
	if containsAny(query, []string{"适合谁"}) {
		terms = append(terms, "适合游客", "游览建议")
	}
	if containsAny(query, []string{"要多久"}) {
		terms = append(terms, "游览时长", "停留时间")
	}
	return terms
}
