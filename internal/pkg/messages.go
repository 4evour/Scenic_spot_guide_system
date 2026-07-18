package pkg

import "github.com/gin-gonic/gin"

// 消息目录 — 集中管理所有用户可见消息的中英文翻译
var messages = map[string]map[string]string{
	"err_bad_request":              {"zh-CN": "参数错误", "en-US": "Bad request"},
	"err_unauthorized":             {"zh-CN": "未登录，请先登录", "en-US": "Please login first"},
	"err_token_expired":            {"zh-CN": "token无效或已过期", "en-US": "Token expired"},
	"err_forbidden":                {"zh-CN": "权限不足，需要管理员权限", "en-US": "Admin permission required"},
	"err_not_found":                {"zh-CN": "资源不存在", "en-US": "Resource not found"},
	"err_internal":                 {"zh-CN": "服务异常", "en-US": "Internal error"},
	"err_csrf":                     {"zh-CN": "CSRF token 校验失败", "en-US": "CSRF token validation failed"},
	"err_ratelimit":                {"zh-CN": "请求过于频繁，请稍后再试", "en-US": "Too many requests. Please wait."},
	"msg_register_success":         {"zh-CN": "注册成功", "en-US": "Registration successful"},
	"msg_register_failed":          {"zh-CN": "注册失败", "en-US": "Registration failed"},
	"msg_login_failed":             {"zh-CN": "用户名或密码错误", "en-US": "Invalid username or password"},
	"msg_token_failed":             {"zh-CN": "生成token失败", "en-US": "Failed to generate token"},
	"msg_logout_success":           {"zh-CN": "已退出登录", "en-US": "Logged out"},
	"msg_update_success":           {"zh-CN": "更新成功", "en-US": "Updated successfully"},
	"msg_update_failed":            {"zh-CN": "更新失败", "en-US": "Update failed"},
	"msg_create_success":           {"zh-CN": "创建成功", "en-US": "Created successfully"},
	"msg_create_failed":            {"zh-CN": "创建失败", "en-US": "Create failed"},
	"msg_delete_success":           {"zh-CN": "删除成功", "en-US": "Deleted successfully"},
	"msg_delete_failed":            {"zh-CN": "删除失败", "en-US": "Delete failed"},
	"msg_save_success":             {"zh-CN": "保存成功", "en-US": "Saved successfully"},
	"msg_save_failed":              {"zh-CN": "保存失败", "en-US": "Save failed"},
	"msg_load_failed":              {"zh-CN": "加载失败", "en-US": "Load failed"},
	"msg_upload_success":           {"zh-CN": "上传成功", "en-US": "Upload successful"},
	"msg_upload_failed":            {"zh-CN": "上传失败", "en-US": "Upload failed"},
	"msg_feedback_thanks":          {"zh-CN": "感谢您的反馈", "en-US": "Thanks for your feedback"},
	"msg_knowledge_uploaded":       {"zh-CN": "知识上传并加载成功", "en-US": "Knowledge uploaded successfully"},
	"msg_knowledge_deleted":        {"zh-CN": "知识删除成功", "en-US": "Knowledge deleted"},
	"msg_knowledge_cleared":        {"zh-CN": "知识清空成功", "en-US": "Knowledge cleared"},
	"msg_ai_failed":                {"zh-CN": "调用AI服务失败", "en-US": "AI service call failed"},
	"msg_scenic_not_found":         {"zh-CN": "景点不存在", "en-US": "Scenic spot not found"},
	"msg_route_not_found":          {"zh-CN": "路线不存在", "en-US": "Route not found"},
	"msg_content_not_found":        {"zh-CN": "讲解内容不存在", "en-US": "Guide content not found"},
	"msg_invalid_id":               {"zh-CN": "无效的ID", "en-US": "Invalid ID"},
	"msg_username_invalid":         {"zh-CN": "用户名须为3-32位字母、数字或下划线", "en-US": "Username must be 3-32 alphanumeric characters or underscores"},
	"msg_email_invalid":            {"zh-CN": "邮箱格式不正确", "en-US": "Invalid email format"},
	"msg_verify_password":          {"zh-CN": "修改邮箱需要验证当前密码", "en-US": "Current password required to change email"},
	"msg_wrong_password":           {"zh-CN": "当前密码错误", "en-US": "Current password is incorrect"},
	"msg_guest_password_forbidden": {"zh-CN": "游客账号不能修改密码，请先注册正式账号", "en-US": "Guest accounts cannot change passwords. Please register first."},
	"msg_password_changed":         {"zh-CN": "密码已修改", "en-US": "Password changed"},
	"msg_no_permission_user":       {"zh-CN": "无权访问该用户信息", "en-US": "Not authorized to access this user's info"},
	"msg_no_permission_modify":     {"zh-CN": "无权修改该用户信息", "en-US": "Not authorized to modify this user"},
	"msg_no_permission_delete":     {"zh-CN": "无权删除该用户", "en-US": "Not authorized to delete this user"},
	"msg_user_not_found":           {"zh-CN": "用户不存在", "en-US": "User not found"},
	"msg_role_required":            {"zh-CN": "角色参数不能为空", "en-US": "Role parameter required"},
	"msg_get_users_failed":         {"zh-CN": "获取用户列表失败", "en-US": "Failed to get user list"},
	"msg_get_spots_failed":         {"zh-CN": "获取景点列表失败", "en-US": "Failed to get scenic spots"},
	"msg_get_routes_failed":        {"zh-CN": "获取路线列表失败", "en-US": "Failed to get routes"},
	"msg_get_content_failed":       {"zh-CN": "获取讲解内容失败", "en-US": "Failed to get guide content"},
	"msg_tts_empty_text":           {"zh-CN": "合成文本不能为空", "en-US": "TTS text cannot be empty"},
	"msg_avatar_not_found":         {"zh-CN": "形象配置不存在", "en-US": "Avatar config not found"},
	"msg_config_not_found":         {"zh-CN": "配置不存在", "en-US": "Configuration not found"},
	"msg_emotion_not_found":        {"zh-CN": "情绪配置未找到", "en-US": "Emotion config not found"},
	// ---- 用户输入相关 ----
	"msg_empty_message":         {"zh-CN": "消息内容不能为空", "en-US": "Message cannot be empty"},
	"msg_empty_input":           {"zh-CN": "输入文本不能为空", "en-US": "Input text cannot be empty"},
	"msg_empty_speech":          {"zh-CN": "语音识别结果不能为空", "en-US": "Speech recognition result cannot be empty"},
	"msg_id_empty":              {"zh-CN": "ID不能为空", "en-US": "ID cannot be empty"},
	"msg_category_required":     {"zh-CN": "分类参数不能为空", "en-US": "Category parameter is required"},
	"msg_content_type_required": {"zh-CN": "内容类型参数不能为空", "en-US": "Content type parameter is required"},
	"msg_difficulty_required":   {"zh-CN": "难度参数不能为空", "en-US": "Difficulty parameter is required"},
	// ---- 景区/路线/内容 CRUD ----
	"msg_create_spot_failed":    {"zh-CN": "创建景点失败", "en-US": "Failed to create scenic spot"},
	"msg_update_spot_failed":    {"zh-CN": "更新景点失败", "en-US": "Failed to update scenic spot"},
	"msg_delete_spot_failed":    {"zh-CN": "删除景点失败", "en-US": "Failed to delete scenic spot"},
	"msg_invalid_spot_id":       {"zh-CN": "无效的景点ID", "en-US": "Invalid scenic spot ID"},
	"msg_create_route_failed":   {"zh-CN": "创建游览路线失败", "en-US": "Failed to create tour route"},
	"msg_update_route_failed":   {"zh-CN": "更新游览路线失败", "en-US": "Failed to update tour route"},
	"msg_delete_route_failed":   {"zh-CN": "删除游览路线失败", "en-US": "Failed to delete tour route"},
	"msg_create_content_failed": {"zh-CN": "创建导览内容失败", "en-US": "Failed to create guide content"},
	"msg_update_content_failed": {"zh-CN": "更新导览内容失败", "en-US": "Failed to update guide content"},
	"msg_delete_content_failed": {"zh-CN": "删除导览内容失败", "en-US": "Failed to delete guide content"},
	// ---- 知识库 ----
	"msg_knowledge_not_init":      {"zh-CN": "知识库服务未初始化", "en-US": "Knowledge base not initialized"},
	"msg_rag_not_init":            {"zh-CN": "RAG服务未初始化", "en-US": "RAG service not initialized"},
	"msg_knowledge_load_failed":   {"zh-CN": "加载知识失败", "en-US": "Failed to load knowledge"},
	"msg_knowledge_query_failed":  {"zh-CN": "查询知识失败", "en-US": "Failed to query knowledge"},
	"msg_knowledge_content_empty": {"zh-CN": "知识内容不能为空", "en-US": "Knowledge content cannot be empty"},
	"msg_knowledge_save_failed":   {"zh-CN": "保存知识失败", "en-US": "Failed to save knowledge"},
	"msg_knowledge_update_failed": {"zh-CN": "更新知识失败", "en-US": "Failed to update knowledge"},
	"msg_knowledge_delete_failed": {"zh-CN": "删除知识失败", "en-US": "Failed to delete knowledge"},
	"msg_knowledge_clear_failed":  {"zh-CN": "清空知识失败", "en-US": "Failed to clear knowledge"},
	"msg_knowledge_not_found":     {"zh-CN": "知识不存在", "en-US": "Knowledge not found"},
	// ---- 文件上传 ----
	"msg_file_get_failed":       {"zh-CN": "获取文件失败", "en-US": "Failed to get file"},
	"msg_file_open_failed":      {"zh-CN": "打开文件失败", "en-US": "Failed to open file"},
	"msg_file_read_failed":      {"zh-CN": "读取文件失败", "en-US": "Failed to read file"},
	"msg_file_save_failed":      {"zh-CN": "保存文件失败", "en-US": "Failed to save file"},
	"msg_file_too_large":        {"zh-CN": "上传文件不能超过 10MB", "en-US": "Upload file must not exceed 10MB"},
	"msg_file_type_unsupported": {"zh-CN": "仅支持 JSONL、JSON、Markdown 或 TXT 文件", "en-US": "Only JSONL, JSON, Markdown, or TXT files are supported"},
	"msg_file_ext_mismatch":     {"zh-CN": "文件内容与扩展名不匹配", "en-US": "File content does not match extension"},
	// ---- 服务状态 ----
	"msg_tts_failed":                  {"zh-CN": "语音合成失败", "en-US": "TTS synthesis failed"},
	"msg_streaming_unsupported":       {"zh-CN": "流式传输不支持", "en-US": "Streaming not supported"},
	"msg_fallback_answer":             {"zh-CN": "抱歉，我暂时无法回答这个问题。", "en-US": "Sorry, I cannot answer this question right now."},
	"msg_service_unavailable":         {"zh-CN": "抱歉，智能服务暂不可用。", "en-US": "Sorry, the smart service is temporarily unavailable."},
	"msg_multimodal_unavailable":      {"zh-CN": "多模态服务暂未启用。", "en-US": "The multimodal service is not enabled."},
	"msg_multimodal_failed":           {"zh-CN": "多模态服务调用失败。", "en-US": "The multimodal service call failed."},
	"msg_multimodal_input_empty":      {"zh-CN": "请提供文字或媒体文件。", "en-US": "Provide text or a media file."},
	"msg_multimodal_message_too_long": {"zh-CN": "文字内容不能超过4000字。", "en-US": "Text cannot exceed 4000 characters."},
	"msg_multimodal_too_many_files":   {"zh-CN": "媒体文件数量不能超过3个。", "en-US": "No more than 3 media files are allowed."},
	"msg_feedback_received":           {"zh-CN": "反馈已接收", "en-US": "Feedback received"},
	// ---- 评估 ----
	"msg_no_eval_data":      {"zh-CN": "暂无评估数据", "en-US": "No evaluation data available"},
	"msg_eval_format_error": {"zh-CN": "评估数据格式错误", "en-US": "Evaluation data format error"},
	// ---- 配置 ----
	"msg_profile_not_loaded": {"zh-CN": "景区配置未加载", "en-US": "Scenic profile not loaded"},
	// ---- 游客问题 ----
	"msg_query_not_found":       {"zh-CN": "游客问题不存在", "en-US": "Visitor query not found"},
	"msg_create_query_failed":   {"zh-CN": "创建游客问题失败", "en-US": "Failed to create visitor query"},
	"msg_get_queries_failed":    {"zh-CN": "获取游客问题列表失败", "en-US": "Failed to get visitor queries"},
	"msg_get_unanswered_failed": {"zh-CN": "获取未回答问题列表失败", "en-US": "Failed to get unanswered queries"},
	"msg_update_query_failed":   {"zh-CN": "更新游客问题失败", "en-US": "Failed to update visitor query"},
	"msg_delete_query_failed":   {"zh-CN": "删除游客问题失败", "en-US": "Failed to delete visitor query"},
}

// fallbackLocale is used when the requested locale is not available
const fallbackLocale = "zh-CN"

// T translates a message key to the user's preferred language.
// Language is detected from the gin context (set by LanguageMiddleware).
func T(c *gin.Context, key string) string {
	lang := c.GetString("lang")
	if lang == "" {
		lang = fallbackLocale
	}
	if entry, ok := messages[key]; ok {
		if msg, ok := entry[lang]; ok {
			return msg
		}
	}
	// Fallback: return the key itself if no translation found
	if entry, ok := messages[key]; ok {
		if msg, ok := entry[fallbackLocale]; ok {
			return msg
		}
	}
	return key
}

// LanguageMiddleware detects the user's preferred language from:
// 1. ?lang= query parameter (highest priority)
// 2. Accept-Language header
// 3. Default: zh-CN
// It stores the result in c.Set("lang", lang).
func LanguageMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Query parameter
		lang := c.Query("lang")
		if lang == "en" || lang == "en-US" {
			lang = "en-US"
		} else if lang == "zh" || lang == "zh-CN" {
			lang = "zh-CN"
		} else if lang != "" {
			lang = "" // unknown value, fall through
		}

		// 2. Accept-Language header
		if lang == "" {
			al := c.GetHeader("Accept-Language")
			if al != "" {
				// Simple parsing: take first language tag
				if len(al) >= 2 {
					prefix := al[:2]
					if prefix == "en" {
						lang = "en-US"
					} else if prefix == "zh" {
						lang = "zh-CN"
					}
				}
			}
		}

		// 3. Default
		if lang == "" {
			lang = fallbackLocale
		}

		c.Set("lang", lang)
		c.Next()
	}
}
