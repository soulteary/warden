// Package i18n 提供了国际化支持功能。
// 支持从请求上下文获取语言，实现多语言文本翻译。
package i18n

import (
	// 标准库
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Language represents the supported languages
type Language string

const (
	// LangEN is English (default)
	LangEN Language = "en"
	// LangZH is Chinese
	LangZH Language = "zh"
	// LangFR is French
	LangFR Language = "fr"
	// LangIT is Italian
	LangIT Language = "it"
	// LangJA is Japanese
	LangJA Language = "ja"
	// LangDE is German
	LangDE Language = "de"
	// LangKO is Korean
	LangKO Language = "ko"
)

// contextKey 用于在上下文中存储语言
type contextKey string

const languageKey contextKey = "language"

// SetLanguageInContext 将语言设置到请求上下文中
func SetLanguageInContext(r *http.Request, lang Language) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), languageKey, lang))
}

// GetLanguageFromContext 从请求上下文中获取语言
func GetLanguageFromContext(r *http.Request) Language {
	if r == nil {
		return LangEN
	}
	if lang, ok := r.Context().Value(languageKey).(Language); ok {
		return lang
	}
	return LangEN
}

// GetLanguageFromContextValue 从 context.Context 中获取语言（用于没有 http.Request 的场景）
func GetLanguageFromContextValue(ctx context.Context) Language {
	if ctx == nil {
		return LangEN
	}
	if lang, ok := ctx.Value(languageKey).(Language); ok {
		return lang
	}
	return LangEN
}

// NormalizeLanguage 规范化语言代码
func NormalizeLanguage(lang string) Language {
	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case "zh", "zh-cn", "zh_cn":
		return LangZH
	case "fr", "fr-fr", "fr_fr":
		return LangFR
	case "it", "it-it", "it_it":
		return LangIT
	case "ja", "ja-jp", "ja_jp":
		return LangJA
	case "de", "de-de", "de_de":
		return LangDE
	case "ko", "ko-kr", "ko_kr":
		return LangKO
	case "en", "en-us", "en_us":
		return LangEN
	default:
		return LangEN
	}
}

// Translations map
var translations = map[Language]map[string]string{
	LangEN: {
		// Error messages
		"error.redis_connection_failed":      "Redis connection failed",
		"error.redis_operation_failed":       "Redis operation failed",
		"error.redis_lock_failed":            "Redis distributed lock operation failed",
		"error.config_load_failed":           "Configuration loading failed",
		"error.config_validation_failed":     "Configuration validation failed",
		"error.config_parse_failed":          "Configuration parsing failed",
		"error.app_init_failed":              "Application initialization failed",
		"error.http_request_failed":          "HTTP request failed",
		"error.http_response_failed":         "HTTP response processing failed",
		"error.data_load_failed":             "Data loading failed",
		"error.data_parse_failed":            "Data parsing failed",
		"error.cache_operation_failed":       "Cache operation failed",
		"error.invalid_parameter":            "Invalid parameter",
		"error.task_execution_failed":        "Task execution failed",
		"error.internal_server_error":        "Internal server error, please try again later",
		"error.not_found":                    "Requested resource does not exist",
		"error.forbidden":                    "Access denied",
		"error.unauthorized":                 "Unauthorized access",
		"error.bad_request":                  "Invalid request parameters",
		"error.too_many_requests":            "Too many requests, please try again later",
		"error.request_failed":               "Request processing failed",
		"error.method_not_allowed":           "Method not allowed",
		"error.user_not_found":               "User not found",
		"error.missing_identifier":           "Bad Request: missing identifier (phone, mail, or user_id)",
		"error.multiple_identifiers":         "Bad Request: only one identifier allowed (phone, mail, or user_id)",
		"error.invalid_pagination":           "Invalid pagination parameters",
		"error.rate_limit_exceeded":          "Rate limit exceeded",
		"error.auth_failed":                  "Authentication failed: invalid API Key",
		"error.api_key_not_configured":       "API Key not configured, request denied (API Key must be configured in production)",
		"error.json_encode_failed":           "JSON encoding failed",
		"error.write_response_failed":        "Failed to write response",
		"error.stream_encode_failed":         "Stream JSON encoding failed",
		"error.encode_error_response_failed": "Failed to encode error response",
		"error.health_check_encode_failed":   "Health check response encoding failed",
		"error.error_response_hidden":        "Error response (details hidden)",
		"error.request_error":                "Error occurred while processing request",

		// Log messages
		"log.http_tls_disabled":                "HTTP TLS certificate verification disabled (development only)",
		"log.prod_tls_required":                "Production environment does not allow disabling TLS certificate verification, exiting",
		"log.redis_password_warning":           "Security warning: Redis password passed via command line argument, recommend using REDIS_PASSWORD or REDIS_PASSWORD_FILE environment variable",
		"log.redis_connection_failed_fallback": "Redis connection failed, falling back to memory mode",
		"log.redis_connected":                  "Redis connection successful",
		"log.redis_disabled":                   "Redis disabled, using memory mode",
		"log.current_mode":                     "Current running mode",
		"log.load_initial_data_failed":         "Failed to load initial data, using empty data",
		"log.check_mode":                       "loadInitialData: checking running mode",
		"log.only_local_detected":              "loadInitialData: ONLY_LOCAL mode detected, skipping remote request",
		"log.loaded_from_local_file":           "Loaded data from local file",
		"log.redis_cache_update_failed":        "Failed to update Redis cache",
		"log.data_file_not_found":              "Data file does not exist",
		"log.only_local_requires_file":         "Tip: ONLY_LOCAL mode requires local data file",
		"log.create_data_file":                 "Please create %s file (refer to %s)",
		"log.only_local_load_failed":           "Local file load failed in ONLY_LOCAL mode, using empty data",
		"log.loaded_from_redis":                "Loaded data from Redis cache",
		"log.loaded_from_remote_api":           "Loaded data from remote API",
		"log.data_file_not_found_no_remote":    "Data file does not exist and remote data address not configured",
		"log.tip_actions":                      "Tip: Please perform one of the following actions:",
		"log.create_data_file_or_config":       "1. Create %s file (refer to %s)",
		"log.config_remote_param":              "2. Or specify remote data address via --config parameter",
		"log.config_remote_env":                "3. Or specify remote data address via CONFIG environment variable",
		"log.using_empty_data":                 "Currently using empty data, service will continue but cannot provide user data",
		"log.all_sources_failed":               "All data sources failed, using empty data",
		"log.retry_redis_cache":                "Retrying Redis cache update",
		"log.redis_cache_updated":              "Redis cache updated",
		"log.background_task_panic":            "Background task panic occurred, recovered",
		"log.data_unchanged":                   "Data unchanged, skipping update",
		"log.redis_cache_failed_continue":      "Failed to update Redis cache, continuing with memory cache",
		"log.data_modified_during_update":      "Data modified during update, skipping Redis update",
		"log.background_update":                "Background data update",
		"log.forced_shutdown":                  "Forced shutdown",
		"log.config_validation_failed_exit":    "Configuration validation failed, exiting",
		"log.app_version":                      "Application version: %s, Build time: %s, Code version: %s",
		"log.scheduler_closed":                 "Scheduled task scheduler closed",
		"log.scheduler_init_failed":            "Scheduled task scheduler initialization failed, exiting",
		"log.service_listening":                "Service listening on port: %s",
		"log.startup_error":                    "Application startup error: %s",
		"log.app_started":                      "Application started successfully",
		"log.shutting_down":                    "Application shutting down, press CTRL+C to exit immediately",
		"log.goodbye":                          "Looking forward to seeing you again",
		"log.unsupported_method":               "Unsupported request method",
		"log.missing_query_params":             "Missing query parameters, need to provide phone, mail, or user_id",
		"log.multiple_query_params":            "Only one query parameter allowed (phone, mail, or user_id)",
		"log.user_not_found":                   "User not found",
		"log.user_query_success":               "User query successful",
		"log.pagination_validation_failed":     "Pagination parameter validation failed",
		"log.request_data_api":                 "Request data API",
		"log.health_check_encode_failed":       "Health check response encoding failed",

		// HTTP response messages
		"http.method_not_allowed":            "Method not allowed",
		"http.user_not_found":                "User not found",
		"http.internal_server_error":         "Internal server error",
		"http.unauthorized":                  "Unauthorized",
		"http.rate_limit_exceeded":           "Rate limit exceeded",
		"http.invalid_pagination_parameters": "Invalid pagination parameters",
	},
	LangZH: {
		// Error messages
		"error.redis_connection_failed":      "Redis 连接失败",
		"error.redis_operation_failed":       "Redis 操作失败",
		"error.redis_lock_failed":            "Redis 分布式锁操作失败",
		"error.config_load_failed":           "配置加载失败",
		"error.config_validation_failed":     "配置验证失败",
		"error.config_parse_failed":          "配置解析失败",
		"error.app_init_failed":              "应用初始化失败",
		"error.http_request_failed":          "HTTP 请求失败",
		"error.http_response_failed":         "HTTP 响应处理失败",
		"error.data_load_failed":             "数据加载失败",
		"error.data_parse_failed":            "数据解析失败",
		"error.cache_operation_failed":       "缓存操作失败",
		"error.invalid_parameter":            "无效的参数",
		"error.task_execution_failed":        "任务执行失败",
		"error.internal_server_error":        "内部服务器错误，请稍后重试",
		"error.not_found":                    "请求的资源不存在",
		"error.forbidden":                    "访问被拒绝",
		"error.unauthorized":                 "未授权访问",
		"error.bad_request":                  "请求参数无效",
		"error.too_many_requests":            "请求过于频繁，请稍后重试",
		"error.request_failed":               "请求处理失败",
		"error.method_not_allowed":           "不支持的请求方法",
		"error.user_not_found":               "用户未找到",
		"error.missing_identifier":           "缺少查询参数，需要提供 phone、mail 或 user_id 之一",
		"error.multiple_identifiers":         "只能提供一个查询参数（phone、mail 或 user_id）",
		"error.invalid_pagination":           "分页参数验证失败",
		"error.rate_limit_exceeded":          "请求过于频繁，请稍后重试",
		"error.auth_failed":                  "认证失败：无效的 API Key",
		"error.api_key_not_configured":       "API Key 未配置，请求被拒绝（生产环境必须配置 API Key）",
		"error.json_encode_failed":           "JSON 编码失败",
		"error.write_response_failed":        "写入响应失败",
		"error.stream_encode_failed":         "流式 JSON 编码失败",
		"error.encode_error_response_failed": "编码错误响应失败",
		"error.health_check_encode_failed":   "健康检查响应编码失败",
		"error.error_response_hidden":        "错误响应（详细信息已隐藏）",
		"error.request_error":                "处理请求时发生错误",

		// Log messages
		"log.http_tls_disabled":                "HTTP TLS 证书验证已禁用（仅用于开发环境）",
		"log.prod_tls_required":                "生产环境不允许禁用 TLS 证书验证，程序退出",
		"log.redis_password_warning":           "⚠️  安全警告：Redis 密码通过命令行参数传递，建议使用环境变量 REDIS_PASSWORD 或 REDIS_PASSWORD_FILE",
		"log.redis_connection_failed_fallback": "⚠️  Redis 连接失败，降级到内存模式（fallback）",
		"log.redis_connected":                  "Redis 连接成功 ✓",
		"log.redis_disabled":                   "Redis 已禁用，使用内存模式",
		"log.current_mode":                     "当前运行模式",
		"log.load_initial_data_failed":         "加载初始数据失败，使用空数据",
		"log.check_mode":                       "loadInitialData: 检查运行模式",
		"log.only_local_detected":              "loadInitialData: 检测到 ONLY_LOCAL 模式，跳过远程请求",
		"log.loaded_from_local_file":           "从本地文件加载数据 ✓",
		"log.redis_cache_update_failed":        "更新 Redis 缓存失败",
		"log.data_file_not_found":              "⚠️  数据文件不存在",
		"log.only_local_requires_file":         "💡 提示：ONLY_LOCAL 模式下需要本地数据文件",
		"log.create_data_file":                 "   请创建 %s 文件（可参考 %s）",
		"log.only_local_load_failed":           "ONLY_LOCAL 模式下本地文件加载失败，使用空数据",
		"log.loaded_from_redis":                "从 Redis 缓存加载数据 ✓",
		"log.loaded_from_remote_api":           "从远程 API 加载数据 ✓",
		"log.data_file_not_found_no_remote":    "⚠️  数据文件不存在且未配置远程数据地址",
		"log.tip_actions":                      "💡 提示：请执行以下操作之一：",
		"log.create_data_file_or_config":       "   1. 创建 %s 文件（可参考 %s）",
		"log.config_remote_param":              "   2. 或通过 --config 参数指定远程数据地址",
		"log.config_remote_env":                "   3. 或通过环境变量 CONFIG 指定远程数据地址",
		"log.using_empty_data":                 "当前使用空数据，服务将继续运行但无法提供用户数据",
		"log.all_sources_failed":               "所有数据源都失败，使用空数据",
		"log.retry_redis_cache":                "重试更新 Redis 缓存",
		"log.redis_cache_updated":              "Redis 缓存已更新",
		"log.background_task_panic":            "后台任务发生 panic，已恢复",
		"log.data_unchanged":                   "数据未变化，跳过更新",
		"log.redis_cache_failed_continue":      "更新 Redis 缓存失败，继续使用内存缓存",
		"log.data_modified_during_update":      "数据在更新过程中被修改，跳过 Redis 更新",
		"log.background_update":                "后台更新数据 📦",
		"log.forced_shutdown":                  "程序强制关闭",
		"log.config_validation_failed_exit":    "配置验证失败，程序退出",
		"log.app_version":                      "程序版本：%s, 构建时间：%s, 代码版本：%s",
		"log.scheduler_closed":                 "定时任务调度器已关闭",
		"log.scheduler_init_failed":            "定时任务调度器初始化失败，程序退出",
		"log.service_listening":                "服务监听端口：%s",
		"log.startup_error":                    "程序启动出错: %s",
		"log.app_started":                      "程序已启动完毕 🚀",
		"log.shutting_down":                    "程序正在关闭中，如需立即结束请按 CTRL+C",
		"log.goodbye":                          "期待与你的再次相遇 ❤️",
		"log.unsupported_method":               "不支持的请求方法",
		"log.missing_query_params":             "缺少查询参数，需要提供 phone、mail 或 user_id 之一",
		"log.multiple_query_params":            "只能提供一个查询参数（phone、mail 或 user_id）",
		"log.user_not_found":                   "用户未找到",
		"log.user_query_success":               "查询用户成功",
		"log.pagination_validation_failed":     "分页参数验证失败",
		"log.request_data_api":                 "请求数据接口 🎩",
		"log.health_check_encode_failed":       "健康检查响应编码失败",

		// HTTP response messages
		"http.method_not_allowed":            "Method not allowed",
		"http.user_not_found":                "User not found",
		"http.internal_server_error":         "Internal server error",
		"http.unauthorized":                  "Unauthorized",
		"http.rate_limit_exceeded":           "Rate limit exceeded",
		"http.invalid_pagination_parameters": "Invalid pagination parameters",
	},
	LangFR: {
		// Error messages
		"error.redis_connection_failed":      "Échec de la connexion Redis",
		"error.redis_operation_failed":       "Échec de l'opération Redis",
		"error.redis_lock_failed":            "Échec de l'opération de verrouillage distribué Redis",
		"error.config_load_failed":           "Échec du chargement de la configuration",
		"error.config_validation_failed":     "Échec de la validation de la configuration",
		"error.config_parse_failed":          "Échec de l'analyse de la configuration",
		"error.app_init_failed":              "Échec de l'initialisation de l'application",
		"error.http_request_failed":          "Échec de la requête HTTP",
		"error.http_response_failed":         "Échec du traitement de la réponse HTTP",
		"error.data_load_failed":             "Échec du chargement des données",
		"error.data_parse_failed":            "Échec de l'analyse des données",
		"error.cache_operation_failed":       "Échec de l'opération de cache",
		"error.invalid_parameter":            "Paramètre invalide",
		"error.task_execution_failed":        "Échec de l'exécution de la tâche",
		"error.internal_server_error":        "Erreur interne du serveur, veuillez réessayer plus tard",
		"error.not_found":                    "La ressource demandée n'existe pas",
		"error.forbidden":                    "Accès refusé",
		"error.unauthorized":                 "Accès non autorisé",
		"error.bad_request":                  "Paramètres de requête invalides",
		"error.too_many_requests":            "Trop de requêtes, veuillez réessayer plus tard",
		"error.request_failed":               "Échec du traitement de la requête",
		"error.method_not_allowed":           "Méthode non autorisée",
		"error.user_not_found":               "Utilisateur non trouvé",
		"error.missing_identifier":           "Requête incorrecte : identifiant manquant (phone, mail ou user_id)",
		"error.multiple_identifiers":         "Requête incorrecte : un seul identifiant autorisé (phone, mail ou user_id)",
		"error.invalid_pagination":           "Paramètres de pagination invalides",
		"error.rate_limit_exceeded":          "Limite de débit dépassée",
		"error.auth_failed":                  "Échec de l'authentification : clé API invalide",
		"error.api_key_not_configured":       "Clé API non configurée, requête refusée (la clé API doit être configurée en production)",
		"error.json_encode_failed":           "Échec de l'encodage JSON",
		"error.write_response_failed":        "Échec de l'écriture de la réponse",
		"error.stream_encode_failed":         "Échec de l'encodage JSON en flux",
		"error.encode_error_response_failed": "Échec de l'encodage de la réponse d'erreur",
		"error.health_check_encode_failed":   "Échec de l'encodage de la réponse de vérification de santé",
		"error.error_response_hidden":        "Réponse d'erreur (détails masqués)",
		"error.request_error":                "Erreur lors du traitement de la requête",

		// Log messages
		"log.http_tls_disabled":                "Vérification du certificat TLS HTTP désactivée (développement uniquement)",
		"log.prod_tls_required":                "L'environnement de production n'autorise pas la désactivation de la vérification du certificat TLS, arrêt",
		"log.redis_password_warning":           "Avertissement de sécurité : mot de passe Redis transmis via argument de ligne de commande, recommande d'utiliser la variable d'environnement REDIS_PASSWORD ou REDIS_PASSWORD_FILE",
		"log.redis_connection_failed_fallback": "Échec de la connexion Redis, basculement en mode mémoire",
		"log.redis_connected":                  "Connexion Redis réussie",
		"log.redis_disabled":                   "Redis désactivé, utilisation du mode mémoire",
		"log.current_mode":                     "Mode d'exécution actuel",
		"log.load_initial_data_failed":         "Échec du chargement des données initiales, utilisation de données vides",
		"log.check_mode":                       "loadInitialData : vérification du mode d'exécution",
		"log.only_local_detected":              "loadInitialData : mode ONLY_LOCAL détecté, saut de la requête distante",
		"log.loaded_from_local_file":           "Données chargées depuis le fichier local",
		"log.redis_cache_update_failed":        "Échec de la mise à jour du cache Redis",
		"log.data_file_not_found":              "Le fichier de données n'existe pas",
		"log.only_local_requires_file":         "Astuce : le mode ONLY_LOCAL nécessite un fichier de données local",
		"log.create_data_file":                 "Veuillez créer le fichier %s (référence : %s)",
		"log.only_local_load_failed":           "Échec du chargement du fichier local en mode ONLY_LOCAL, utilisation de données vides",
		"log.loaded_from_redis":                "Données chargées depuis le cache Redis",
		"log.loaded_from_remote_api":           "Données chargées depuis l'API distante",
		"log.data_file_not_found_no_remote":    "Le fichier de données n'existe pas et l'adresse de données distante n'est pas configurée",
		"log.tip_actions":                      "Astuce : veuillez effectuer l'une des actions suivantes :",
		"log.create_data_file_or_config":       "1. Créer le fichier %s (référence : %s)",
		"log.config_remote_param":              "2. Ou spécifier l'adresse de données distante via le paramètre --config",
		"log.config_remote_env":                "3. Ou spécifier l'adresse de données distante via la variable d'environnement CONFIG",
		"log.using_empty_data":                 "Utilisation actuelle de données vides, le service continuera mais ne pourra pas fournir de données utilisateur",
		"log.all_sources_failed":               "Toutes les sources de données ont échoué, utilisation de données vides",
		"log.retry_redis_cache":                "Nouvelle tentative de mise à jour du cache Redis",
		"log.redis_cache_updated":              "Cache Redis mis à jour",
		"log.background_task_panic":            "Panique de la tâche en arrière-plan survenue, récupérée",
		"log.data_unchanged":                   "Données inchangées, saut de la mise à jour",
		"log.redis_cache_failed_continue":      "Échec de la mise à jour du cache Redis, continuation avec le cache mémoire",
		"log.data_modified_during_update":      "Données modifiées pendant la mise à jour, saut de la mise à jour Redis",
		"log.background_update":                "Mise à jour des données en arrière-plan",
		"log.forced_shutdown":                  "Arrêt forcé",
		"log.config_validation_failed_exit":    "Échec de la validation de la configuration, arrêt",
		"log.app_version":                      "Version de l'application : %s, Heure de construction : %s, Version du code : %s",
		"log.scheduler_closed":                 "Planificateur de tâches planifiées fermé",
		"log.scheduler_init_failed":            "Échec de l'initialisation du planificateur de tâches planifiées, arrêt",
		"log.service_listening":                "Service à l'écoute sur le port : %s",
		"log.startup_error":                    "Erreur de démarrage de l'application : %s",
		"log.app_started":                      "Application démarrée avec succès",
		"log.shutting_down":                    "Application en cours d'arrêt, appuyez sur CTRL+C pour quitter immédiatement",
		"log.goodbye":                          "Au revoir",
		"log.unsupported_method":               "Méthode de requête non prise en charge",
		"log.missing_query_params":             "Paramètres de requête manquants, besoin de fournir phone, mail ou user_id",
		"log.multiple_query_params":            "Un seul paramètre de requête autorisé (phone, mail ou user_id)",
		"log.user_not_found":                   "Utilisateur non trouvé",
		"log.user_query_success":               "Requête utilisateur réussie",
		"log.pagination_validation_failed":     "Échec de la validation des paramètres de pagination",
		"log.request_data_api":                 "Requête API de données",
		"log.health_check_encode_failed":       "Échec de l'encodage de la réponse de vérification de santé",

		// HTTP response messages
		"http.method_not_allowed":            "Method not allowed",
		"http.user_not_found":                "User not found",
		"http.internal_server_error":         "Internal server error",
		"http.unauthorized":                  "Unauthorized",
		"http.rate_limit_exceeded":           "Rate limit exceeded",
		"http.invalid_pagination_parameters": "Invalid pagination parameters",
	},
	LangIT: {
		// Error messages
		"error.redis_connection_failed":      "Connessione Redis fallita",
		"error.redis_operation_failed":       "Operazione Redis fallita",
		"error.redis_lock_failed":            "Operazione di blocco distribuito Redis fallita",
		"error.config_load_failed":           "Caricamento configurazione fallito",
		"error.config_validation_failed":     "Validazione configurazione fallita",
		"error.config_parse_failed":          "Analisi configurazione fallita",
		"error.app_init_failed":              "Inizializzazione applicazione fallita",
		"error.http_request_failed":          "Richiesta HTTP fallita",
		"error.http_response_failed":         "Elaborazione risposta HTTP fallita",
		"error.data_load_failed":             "Caricamento dati fallito",
		"error.data_parse_failed":            "Analisi dati fallita",
		"error.cache_operation_failed":       "Operazione cache fallita",
		"error.invalid_parameter":            "Parametro non valido",
		"error.task_execution_failed":        "Esecuzione attività fallita",
		"error.internal_server_error":        "Errore interno del server, riprovare più tardi",
		"error.not_found":                    "La risorsa richiesta non esiste",
		"error.forbidden":                    "Accesso negato",
		"error.unauthorized":                 "Accesso non autorizzato",
		"error.bad_request":                  "Parametri richiesta non validi",
		"error.too_many_requests":            "Troppe richieste, riprovare più tardi",
		"error.request_failed":               "Elaborazione richiesta fallita",
		"error.method_not_allowed":           "Metodo non consentito",
		"error.user_not_found":               "Utente non trovato",
		"error.missing_identifier":           "Richiesta errata: identificatore mancante (phone, mail o user_id)",
		"error.multiple_identifiers":         "Richiesta errata: è consentito un solo identificatore (phone, mail o user_id)",
		"error.invalid_pagination":           "Parametri di paginazione non validi",
		"error.rate_limit_exceeded":          "Limite di velocità superato",
		"error.auth_failed":                  "Autenticazione fallita: chiave API non valida",
		"error.api_key_not_configured":       "Chiave API non configurata, richiesta negata (la chiave API deve essere configurata in produzione)",
		"error.json_encode_failed":           "Codifica JSON fallita",
		"error.write_response_failed":        "Scrittura risposta fallita",
		"error.stream_encode_failed":         "Codifica JSON in streaming fallita",
		"error.encode_error_response_failed": "Codifica risposta di errore fallita",
		"error.health_check_encode_failed":   "Codifica risposta controllo salute fallita",
		"error.error_response_hidden":        "Risposta di errore (dettagli nascosti)",
		"error.request_error":                "Errore durante l'elaborazione della richiesta",

		// Log messages
		"log.http_tls_disabled":                "Verifica certificato TLS HTTP disabilitata (solo sviluppo)",
		"log.prod_tls_required":                "L'ambiente di produzione non consente la disabilitazione della verifica del certificato TLS, uscita",
		"log.redis_password_warning":           "Avviso di sicurezza: password Redis passata tramite argomento della riga di comando, si consiglia di utilizzare la variabile d'ambiente REDIS_PASSWORD o REDIS_PASSWORD_FILE",
		"log.redis_connection_failed_fallback": "Connessione Redis fallita, passaggio alla modalità memoria",
		"log.redis_connected":                  "Connessione Redis riuscita",
		"log.redis_disabled":                   "Redis disabilitato, utilizzo della modalità memoria",
		"log.current_mode":                     "Modalità di esecuzione corrente",
		"log.load_initial_data_failed":         "Caricamento dati iniziali fallito, utilizzo di dati vuoti",
		"log.check_mode":                       "loadInitialData: verifica modalità di esecuzione",
		"log.only_local_detected":              "loadInitialData: modalità ONLY_LOCAL rilevata, salto della richiesta remota",
		"log.loaded_from_local_file":           "Dati caricati dal file locale",
		"log.redis_cache_update_failed":        "Aggiornamento cache Redis fallito",
		"log.data_file_not_found":              "Il file di dati non esiste",
		"log.only_local_requires_file":         "Suggerimento: la modalità ONLY_LOCAL richiede un file di dati locale",
		"log.create_data_file":                 "Creare il file %s (riferimento: %s)",
		"log.only_local_load_failed":           "Caricamento file locale fallito in modalità ONLY_LOCAL, utilizzo di dati vuoti",
		"log.loaded_from_redis":                "Dati caricati dalla cache Redis",
		"log.loaded_from_remote_api":           "Dati caricati dall'API remota",
		"log.data_file_not_found_no_remote":    "Il file di dati non esiste e l'indirizzo dati remoto non è configurato",
		"log.tip_actions":                      "Suggerimento: eseguire una delle seguenti azioni:",
		"log.create_data_file_or_config":       "1. Creare il file %s (riferimento: %s)",
		"log.config_remote_param":              "2. O specificare l'indirizzo dati remoto tramite il parametro --config",
		"log.config_remote_env":                "3. O specificare l'indirizzo dati remoto tramite la variabile d'ambiente CONFIG",
		"log.using_empty_data":                 "Utilizzo attuale di dati vuoti, il servizio continuerà ma non potrà fornire dati utente",
		"log.all_sources_failed":               "Tutte le fonti di dati sono fallite, utilizzo di dati vuoti",
		"log.retry_redis_cache":                "Riprovare l'aggiornamento della cache Redis",
		"log.redis_cache_updated":              "Cache Redis aggiornata",
		"log.background_task_panic":            "Panico dell'attività in background verificatosi, recuperato",
		"log.data_unchanged":                   "Dati invariati, salto dell'aggiornamento",
		"log.redis_cache_failed_continue":      "Aggiornamento cache Redis fallito, continuazione con cache memoria",
		"log.data_modified_during_update":      "Dati modificati durante l'aggiornamento, salto dell'aggiornamento Redis",
		"log.background_update":                "Aggiornamento dati in background",
		"log.forced_shutdown":                  "Arresto forzato",
		"log.config_validation_failed_exit":    "Validazione configurazione fallita, uscita",
		"log.app_version":                      "Versione applicazione: %s, Ora di compilazione: %s, Versione codice: %s",
		"log.scheduler_closed":                 "Pianificatore attività pianificate chiuso",
		"log.scheduler_init_failed":            "Inizializzazione pianificatore attività pianificate fallita, uscita",
		"log.service_listening":                "Servizio in ascolto sulla porta: %s",
		"log.startup_error":                    "Errore di avvio applicazione: %s",
		"log.app_started":                      "Applicazione avviata con successo",
		"log.shutting_down":                    "Applicazione in chiusura, premere CTRL+C per uscire immediatamente",
		"log.goodbye":                          "Arrivederci",
		"log.unsupported_method":               "Metodo di richiesta non supportato",
		"log.missing_query_params":             "Parametri di richiesta mancanti, necessario fornire phone, mail o user_id",
		"log.multiple_query_params":            "È consentito un solo parametro di richiesta (phone, mail o user_id)",
		"log.user_not_found":                   "Utente non trovato",
		"log.user_query_success":               "Richiesta utente riuscita",
		"log.pagination_validation_failed":     "Validazione parametri di paginazione fallita",
		"log.request_data_api":                 "Richiesta API dati",
		"log.health_check_encode_failed":       "Codifica risposta controllo salute fallita",

		// HTTP response messages
		"http.method_not_allowed":            "Method not allowed",
		"http.user_not_found":                "User not found",
		"http.internal_server_error":         "Internal server error",
		"http.unauthorized":                  "Unauthorized",
		"http.rate_limit_exceeded":           "Rate limit exceeded",
		"http.invalid_pagination_parameters": "Invalid pagination parameters",
	},
	LangJA: {
		// Error messages
		"error.redis_connection_failed":      "Redis接続に失敗しました",
		"error.redis_operation_failed":       "Redis操作に失敗しました",
		"error.redis_lock_failed":            "Redis分散ロック操作に失敗しました",
		"error.config_load_failed":           "設定の読み込みに失敗しました",
		"error.config_validation_failed":     "設定の検証に失敗しました",
		"error.config_parse_failed":          "設定の解析に失敗しました",
		"error.app_init_failed":              "アプリケーションの初期化に失敗しました",
		"error.http_request_failed":          "HTTPリクエストに失敗しました",
		"error.http_response_failed":         "HTTPレスポンスの処理に失敗しました",
		"error.data_load_failed":             "データの読み込みに失敗しました",
		"error.data_parse_failed":            "データの解析に失敗しました",
		"error.cache_operation_failed":       "キャッシュ操作に失敗しました",
		"error.invalid_parameter":            "無効なパラメータ",
		"error.task_execution_failed":        "タスクの実行に失敗しました",
		"error.internal_server_error":        "内部サーバーエラー、後でもう一度お試しください",
		"error.not_found":                    "要求されたリソースが存在しません",
		"error.forbidden":                    "アクセスが拒否されました",
		"error.unauthorized":                 "認証されていないアクセス",
		"error.bad_request":                  "リクエストパラメータが無効です",
		"error.too_many_requests":            "リクエストが多すぎます。後でもう一度お試しください",
		"error.request_failed":               "リクエストの処理に失敗しました",
		"error.method_not_allowed":           "許可されていないメソッド",
		"error.user_not_found":               "ユーザーが見つかりません",
		"error.missing_identifier":           "不正なリクエスト：識別子が不足しています（phone、mail、またはuser_id）",
		"error.multiple_identifiers":         "不正なリクエスト：識別子は1つだけ許可されています（phone、mail、またはuser_id）",
		"error.invalid_pagination":           "ページネーションパラメータが無効です",
		"error.rate_limit_exceeded":          "レート制限を超えました",
		"error.auth_failed":                  "認証に失敗しました：無効なAPIキー",
		"error.api_key_not_configured":       "APIキーが設定されていません。リクエストが拒否されました（本番環境ではAPIキーを設定する必要があります）",
		"error.json_encode_failed":           "JSONエンコードに失敗しました",
		"error.write_response_failed":        "レスポンスの書き込みに失敗しました",
		"error.stream_encode_failed":         "ストリームJSONエンコードに失敗しました",
		"error.encode_error_response_failed": "エラーレスポンスのエンコードに失敗しました",
		"error.health_check_encode_failed":   "ヘルスチェックレスポンスのエンコードに失敗しました",
		"error.error_response_hidden":        "エラーレスポンス（詳細は非表示）",
		"error.request_error":                "リクエストの処理中にエラーが発生しました",

		// Log messages
		"log.http_tls_disabled":                "HTTP TLS証明書の検証が無効になっています（開発環境のみ）",
		"log.prod_tls_required":                "本番環境ではTLS証明書の検証を無効にすることはできません。終了します",
		"log.redis_password_warning":           "セキュリティ警告：Redisパスワードがコマンドライン引数経由で渡されました。REDIS_PASSWORDまたはREDIS_PASSWORD_FILE環境変数の使用を推奨します",
		"log.redis_connection_failed_fallback": "Redis接続に失敗しました。メモリモードにフォールバックします",
		"log.redis_connected":                  "Redis接続に成功しました",
		"log.redis_disabled":                   "Redisが無効になっています。メモリモードを使用します",
		"log.current_mode":                     "現在の実行モード",
		"log.load_initial_data_failed":         "初期データの読み込みに失敗しました。空のデータを使用します",
		"log.check_mode":                       "loadInitialData：実行モードを確認中",
		"log.only_local_detected":              "loadInitialData：ONLY_LOCALモードが検出されました。リモートリクエストをスキップします",
		"log.loaded_from_local_file":           "ローカルファイルからデータを読み込みました",
		"log.redis_cache_update_failed":        "Redisキャッシュの更新に失敗しました",
		"log.data_file_not_found":              "データファイルが存在しません",
		"log.only_local_requires_file":         "ヒント：ONLY_LOCALモードにはローカルデータファイルが必要です",
		"log.create_data_file":                 "%sファイルを作成してください（参照：%s）",
		"log.only_local_load_failed":           "ONLY_LOCALモードでローカルファイルの読み込みに失敗しました。空のデータを使用します",
		"log.loaded_from_redis":                "Redisキャッシュからデータを読み込みました",
		"log.loaded_from_remote_api":           "リモートAPIからデータを読み込みました",
		"log.data_file_not_found_no_remote":    "データファイルが存在せず、リモートデータアドレスが設定されていません",
		"log.tip_actions":                      "ヒント：次のいずれかの操作を実行してください：",
		"log.create_data_file_or_config":       "1. %sファイルを作成する（参照：%s）",
		"log.config_remote_param":              "2. または--configパラメータでリモートデータアドレスを指定する",
		"log.config_remote_env":                "3. またはCONFIG環境変数でリモートデータアドレスを指定する",
		"log.using_empty_data":                 "現在空のデータを使用しています。サービスは継続しますが、ユーザーデータを提供できません",
		"log.all_sources_failed":               "すべてのデータソースが失敗しました。空のデータを使用します",
		"log.retry_redis_cache":                "Redisキャッシュの更新を再試行中",
		"log.redis_cache_updated":              "Redisキャッシュが更新されました",
		"log.background_task_panic":            "バックグラウンドタスクでパニックが発生しました。回復しました",
		"log.data_unchanged":                   "データに変更がありません。更新をスキップします",
		"log.redis_cache_failed_continue":      "Redisキャッシュの更新に失敗しました。メモリキャッシュを続行します",
		"log.data_modified_during_update":      "更新中にデータが変更されました。Redis更新をスキップします",
		"log.background_update":                "バックグラウンドデータ更新",
		"log.forced_shutdown":                  "強制シャットダウン",
		"log.config_validation_failed_exit":    "設定の検証に失敗しました。終了します",
		"log.app_version":                      "アプリケーションバージョン：%s、ビルド時刻：%s、コードバージョン：%s",
		"log.scheduler_closed":                 "スケジュールされたタスクスケジューラーが閉じられました",
		"log.scheduler_init_failed":            "スケジュールされたタスクスケジューラーの初期化に失敗しました。終了します",
		"log.service_listening":                "サービスがポートでリッスン中：%s",
		"log.startup_error":                    "アプリケーションの起動エラー：%s",
		"log.app_started":                      "アプリケーションが正常に起動しました",
		"log.shutting_down":                    "アプリケーションをシャットダウン中です。すぐに終了するにはCTRL+Cを押してください",
		"log.goodbye":                          "またお会いしましょう",
		"log.unsupported_method":               "サポートされていないリクエストメソッド",
		"log.missing_query_params":             "クエリパラメータが不足しています。phone、mail、またはuser_idを提供する必要があります",
		"log.multiple_query_params":            "1つのクエリパラメータのみ許可されています（phone、mail、またはuser_id）",
		"log.user_not_found":                   "ユーザーが見つかりません",
		"log.user_query_success":               "ユーザークエリが成功しました",
		"log.pagination_validation_failed":     "ページネーションパラメータの検証に失敗しました",
		"log.request_data_api":                 "データAPIをリクエスト",
		"log.health_check_encode_failed":       "ヘルスチェックレスポンスのエンコードに失敗しました",

		// HTTP response messages
		"http.method_not_allowed":            "Method not allowed",
		"http.user_not_found":                "User not found",
		"http.internal_server_error":         "Internal server error",
		"http.unauthorized":                  "Unauthorized",
		"http.rate_limit_exceeded":           "Rate limit exceeded",
		"http.invalid_pagination_parameters": "Invalid pagination parameters",
	},
	LangDE: {
		// Error messages
		"error.redis_connection_failed":      "Redis-Verbindung fehlgeschlagen",
		"error.redis_operation_failed":       "Redis-Operation fehlgeschlagen",
		"error.redis_lock_failed":            "Redis-Verteilte Sperr-Operation fehlgeschlagen",
		"error.config_load_failed":           "Konfiguration konnte nicht geladen werden",
		"error.config_validation_failed":     "Konfigurationsvalidierung fehlgeschlagen",
		"error.config_parse_failed":          "Konfigurationsanalyse fehlgeschlagen",
		"error.app_init_failed":              "Anwendungsinitialisierung fehlgeschlagen",
		"error.http_request_failed":          "HTTP-Anfrage fehlgeschlagen",
		"error.http_response_failed":         "HTTP-Antwortverarbeitung fehlgeschlagen",
		"error.data_load_failed":             "Daten konnten nicht geladen werden",
		"error.data_parse_failed":            "Datenanalyse fehlgeschlagen",
		"error.cache_operation_failed":       "Cache-Operation fehlgeschlagen",
		"error.invalid_parameter":            "Ungültiger Parameter",
		"error.task_execution_failed":        "Aufgabenausführung fehlgeschlagen",
		"error.internal_server_error":        "Interner Serverfehler, bitte versuchen Sie es später erneut",
		"error.not_found":                    "Angeforderte Ressource existiert nicht",
		"error.forbidden":                    "Zugriff verweigert",
		"error.unauthorized":                 "Nicht autorisierter Zugriff",
		"error.bad_request":                  "Ungültige Anfrageparameter",
		"error.too_many_requests":            "Zu viele Anfragen, bitte versuchen Sie es später erneut",
		"error.request_failed":               "Anfrageverarbeitung fehlgeschlagen",
		"error.method_not_allowed":           "Methode nicht erlaubt",
		"error.user_not_found":               "Benutzer nicht gefunden",
		"error.missing_identifier":           "Ungültige Anfrage: Identifikator fehlt (phone, mail oder user_id)",
		"error.multiple_identifiers":         "Ungültige Anfrage: nur ein Identifikator erlaubt (phone, mail oder user_id)",
		"error.invalid_pagination":           "Ungültige Paginierungsparameter",
		"error.rate_limit_exceeded":          "Rate-Limit überschritten",
		"error.auth_failed":                  "Authentifizierung fehlgeschlagen: ungültiger API-Schlüssel",
		"error.api_key_not_configured":       "API-Schlüssel nicht konfiguriert, Anfrage abgelehnt (API-Schlüssel muss in der Produktion konfiguriert werden)",
		"error.json_encode_failed":           "JSON-Kodierung fehlgeschlagen",
		"error.write_response_failed":        "Antwort konnte nicht geschrieben werden",
		"error.stream_encode_failed":         "Stream-JSON-Kodierung fehlgeschlagen",
		"error.encode_error_response_failed": "Fehlerantwort konnte nicht kodiert werden",
		"error.health_check_encode_failed":   "Health-Check-Antwort konnte nicht kodiert werden",
		"error.error_response_hidden":        "Fehlerantwort (Details ausgeblendet)",
		"error.request_error":                "Fehler bei der Verarbeitung der Anfrage",

		// Log messages
		"log.http_tls_disabled":                "HTTP TLS-Zertifikatsüberprüfung deaktiviert (nur Entwicklung)",
		"log.prod_tls_required":                "Produktionsumgebung erlaubt keine Deaktivierung der TLS-Zertifikatsüberprüfung, Beendigung",
		"log.redis_password_warning":           "Sicherheitswarnung: Redis-Passwort über Befehlszeilenargument übergeben, empfohlen wird die Verwendung der Umgebungsvariable REDIS_PASSWORD oder REDIS_PASSWORD_FILE",
		"log.redis_connection_failed_fallback": "Redis-Verbindung fehlgeschlagen, Fallback auf Speichermodus",
		"log.redis_connected":                  "Redis-Verbindung erfolgreich",
		"log.redis_disabled":                   "Redis deaktiviert, Verwendung des Speichermodus",
		"log.current_mode":                     "Aktueller Ausführungsmodus",
		"log.load_initial_data_failed":         "Laden der Anfangsdaten fehlgeschlagen, Verwendung leerer Daten",
		"log.check_mode":                       "loadInitialData: Überprüfung des Ausführungsmodus",
		"log.only_local_detected":              "loadInitialData: ONLY_LOCAL-Modus erkannt, Überspringen der Remote-Anfrage",
		"log.loaded_from_local_file":           "Daten aus lokaler Datei geladen",
		"log.redis_cache_update_failed":        "Redis-Cache-Update fehlgeschlagen",
		"log.data_file_not_found":              "Datendatei existiert nicht",
		"log.only_local_requires_file":         "Hinweis: ONLY_LOCAL-Modus erfordert lokale Datendatei",
		"log.create_data_file":                 "Bitte erstellen Sie die Datei %s (Referenz: %s)",
		"log.only_local_load_failed":           "Laden der lokalen Datei im ONLY_LOCAL-Modus fehlgeschlagen, Verwendung leerer Daten",
		"log.loaded_from_redis":                "Daten aus Redis-Cache geladen",
		"log.loaded_from_remote_api":           "Daten aus Remote-API geladen",
		"log.data_file_not_found_no_remote":    "Datendatei existiert nicht und Remote-Datenadresse nicht konfiguriert",
		"log.tip_actions":                      "Hinweis: Bitte führen Sie eine der folgenden Aktionen aus:",
		"log.create_data_file_or_config":       "1. Erstellen Sie die Datei %s (Referenz: %s)",
		"log.config_remote_param":              "2. Oder geben Sie die Remote-Datenadresse über den Parameter --config an",
		"log.config_remote_env":                "3. Oder geben Sie die Remote-Datenadresse über die Umgebungsvariable CONFIG an",
		"log.using_empty_data":                 "Aktuell werden leere Daten verwendet, der Dienst wird fortgesetzt, kann aber keine Benutzerdaten bereitstellen",
		"log.all_sources_failed":               "Alle Datenquellen fehlgeschlagen, Verwendung leerer Daten",
		"log.retry_redis_cache":                "Wiederholung des Redis-Cache-Updates",
		"log.redis_cache_updated":              "Redis-Cache aktualisiert",
		"log.background_task_panic":            "Panik bei Hintergrundaufgabe aufgetreten, wiederhergestellt",
		"log.data_unchanged":                   "Daten unverändert, Überspringen der Aktualisierung",
		"log.redis_cache_failed_continue":      "Redis-Cache-Update fehlgeschlagen, Fortsetzung mit Speicher-Cache",
		"log.data_modified_during_update":      "Daten während der Aktualisierung geändert, Überspringen der Redis-Aktualisierung",
		"log.background_update":                "Hintergrund-Datenaktualisierung",
		"log.forced_shutdown":                  "Erzwungener Shutdown",
		"log.config_validation_failed_exit":    "Konfigurationsvalidierung fehlgeschlagen, Beendigung",
		"log.app_version":                      "Anwendungsversion: %s, Build-Zeit: %s, Code-Version: %s",
		"log.scheduler_closed":                 "Geplanter Aufgabenplaner geschlossen",
		"log.scheduler_init_failed":            "Initialisierung des geplanten Aufgabenplaners fehlgeschlagen, Beendigung",
		"log.service_listening":                "Dienst lauscht auf Port: %s",
		"log.startup_error":                    "Anwendungsstartfehler: %s",
		"log.app_started":                      "Anwendung erfolgreich gestartet",
		"log.shutting_down":                    "Anwendung wird heruntergefahren, drücken Sie CTRL+C, um sofort zu beenden",
		"log.goodbye":                          "Auf Wiedersehen",
		"log.unsupported_method":               "Nicht unterstützte Anfragemethode",
		"log.missing_query_params":             "Fehlende Abfrageparameter, müssen phone, mail oder user_id bereitstellen",
		"log.multiple_query_params":            "Nur ein Abfrageparameter erlaubt (phone, mail oder user_id)",
		"log.user_not_found":                   "Benutzer nicht gefunden",
		"log.user_query_success":               "Benutzerabfrage erfolgreich",
		"log.pagination_validation_failed":     "Validierung der Paginierungsparameter fehlgeschlagen",
		"log.request_data_api":                 "Daten-API anfordern",
		"log.health_check_encode_failed":       "Health-Check-Antwort konnte nicht kodiert werden",

		// HTTP response messages
		"http.method_not_allowed":            "Method not allowed",
		"http.user_not_found":                "User not found",
		"http.internal_server_error":         "Internal server error",
		"http.unauthorized":                  "Unauthorized",
		"http.rate_limit_exceeded":           "Rate limit exceeded",
		"http.invalid_pagination_parameters": "Invalid pagination parameters",
	},
	LangKO: {
		// Error messages
		"error.redis_connection_failed":      "Redis 연결 실패",
		"error.redis_operation_failed":       "Redis 작업 실패",
		"error.redis_lock_failed":            "Redis 분산 잠금 작업 실패",
		"error.config_load_failed":           "구성 로드 실패",
		"error.config_validation_failed":     "구성 유효성 검사 실패",
		"error.config_parse_failed":          "구성 구문 분석 실패",
		"error.app_init_failed":              "애플리케이션 초기화 실패",
		"error.http_request_failed":          "HTTP 요청 실패",
		"error.http_response_failed":         "HTTP 응답 처리 실패",
		"error.data_load_failed":             "데이터 로드 실패",
		"error.data_parse_failed":            "데이터 구문 분석 실패",
		"error.cache_operation_failed":       "캐시 작업 실패",
		"error.invalid_parameter":            "잘못된 매개변수",
		"error.task_execution_failed":        "작업 실행 실패",
		"error.internal_server_error":        "내부 서버 오류, 나중에 다시 시도하세요",
		"error.not_found":                    "요청한 리소스가 존재하지 않습니다",
		"error.forbidden":                    "액세스가 거부되었습니다",
		"error.unauthorized":                 "인증되지 않은 액세스",
		"error.bad_request":                  "잘못된 요청 매개변수",
		"error.too_many_requests":            "요청이 너무 많습니다. 나중에 다시 시도하세요",
		"error.request_failed":               "요청 처리 실패",
		"error.method_not_allowed":           "허용되지 않은 메서드",
		"error.user_not_found":               "사용자를 찾을 수 없습니다",
		"error.missing_identifier":           "잘못된 요청: 식별자가 없습니다 (phone, mail 또는 user_id)",
		"error.multiple_identifiers":         "잘못된 요청: 하나의 식별자만 허용됩니다 (phone, mail 또는 user_id)",
		"error.invalid_pagination":           "잘못된 페이지 매김 매개변수",
		"error.rate_limit_exceeded":          "속도 제한 초과",
		"error.auth_failed":                  "인증 실패: 잘못된 API 키",
		"error.api_key_not_configured":       "API 키가 구성되지 않았습니다. 요청이 거부되었습니다 (프로덕션에서는 API 키를 구성해야 합니다)",
		"error.json_encode_failed":           "JSON 인코딩 실패",
		"error.write_response_failed":        "응답 쓰기 실패",
		"error.stream_encode_failed":         "스트림 JSON 인코딩 실패",
		"error.encode_error_response_failed": "오류 응답 인코딩 실패",
		"error.health_check_encode_failed":   "상태 확인 응답 인코딩 실패",
		"error.error_response_hidden":        "오류 응답 (세부 정보 숨김)",
		"error.request_error":                "요청 처리 중 오류 발생",

		// Log messages
		"log.http_tls_disabled":                "HTTP TLS 인증서 확인이 비활성화되었습니다 (개발 전용)",
		"log.prod_tls_required":                "프로덕션 환경에서는 TLS 인증서 확인을 비활성화할 수 없습니다. 종료합니다",
		"log.redis_password_warning":           "보안 경고: Redis 비밀번호가 명령줄 인수를 통해 전달되었습니다. REDIS_PASSWORD 또는 REDIS_PASSWORD_FILE 환경 변수 사용을 권장합니다",
		"log.redis_connection_failed_fallback": "Redis 연결 실패, 메모리 모드로 폴백",
		"log.redis_connected":                  "Redis 연결 성공",
		"log.redis_disabled":                   "Redis가 비활성화되었습니다. 메모리 모드를 사용합니다",
		"log.current_mode":                     "현재 실행 모드",
		"log.load_initial_data_failed":         "초기 데이터 로드 실패, 빈 데이터 사용",
		"log.check_mode":                       "loadInitialData: 실행 모드 확인",
		"log.only_local_detected":              "loadInitialData: ONLY_LOCAL 모드 감지, 원격 요청 건너뛰기",
		"log.loaded_from_local_file":           "로컬 파일에서 데이터 로드됨",
		"log.redis_cache_update_failed":        "Redis 캐시 업데이트 실패",
		"log.data_file_not_found":              "데이터 파일이 존재하지 않습니다",
		"log.only_local_requires_file":         "팁: ONLY_LOCAL 모드에는 로컬 데이터 파일이 필요합니다",
		"log.create_data_file":                 "%s 파일을 만드세요 (참조: %s)",
		"log.only_local_load_failed":           "ONLY_LOCAL 모드에서 로컬 파일 로드 실패, 빈 데이터 사용",
		"log.loaded_from_redis":                "Redis 캐시에서 데이터 로드됨",
		"log.loaded_from_remote_api":           "원격 API에서 데이터 로드됨",
		"log.data_file_not_found_no_remote":    "데이터 파일이 존재하지 않으며 원격 데이터 주소가 구성되지 않았습니다",
		"log.tip_actions":                      "팁: 다음 작업 중 하나를 수행하세요:",
		"log.create_data_file_or_config":       "1. %s 파일 만들기 (참조: %s)",
		"log.config_remote_param":              "2. 또는 --config 매개변수를 통해 원격 데이터 주소 지정",
		"log.config_remote_env":                "3. 또는 CONFIG 환경 변수를 통해 원격 데이터 주소 지정",
		"log.using_empty_data":                 "현재 빈 데이터를 사용하고 있습니다. 서비스는 계속되지만 사용자 데이터를 제공할 수 없습니다",
		"log.all_sources_failed":               "모든 데이터 소스 실패, 빈 데이터 사용",
		"log.retry_redis_cache":                "Redis 캐시 업데이트 재시도",
		"log.redis_cache_updated":              "Redis 캐시 업데이트됨",
		"log.background_task_panic":            "백그라운드 작업에서 패닉 발생, 복구됨",
		"log.data_unchanged":                   "데이터 변경 없음, 업데이트 건너뛰기",
		"log.redis_cache_failed_continue":      "Redis 캐시 업데이트 실패, 메모리 캐시 계속",
		"log.data_modified_during_update":      "업데이트 중 데이터 수정됨, Redis 업데이트 건너뛰기",
		"log.background_update":                "백그라운드 데이터 업데이트",
		"log.forced_shutdown":                  "강제 종료",
		"log.config_validation_failed_exit":    "구성 유효성 검사 실패, 종료",
		"log.app_version":                      "애플리케이션 버전: %s, 빌드 시간: %s, 코드 버전: %s",
		"log.scheduler_closed":                 "예약된 작업 스케줄러 닫힘",
		"log.scheduler_init_failed":            "예약된 작업 스케줄러 초기화 실패, 종료",
		"log.service_listening":                "서비스가 포트에서 수신 중: %s",
		"log.startup_error":                    "애플리케이션 시작 오류: %s",
		"log.app_started":                      "애플리케이션이 성공적으로 시작되었습니다",
		"log.shutting_down":                    "애플리케이션 종료 중입니다. 즉시 종료하려면 CTRL+C를 누르세요",
		"log.goodbye":                          "다시 만나기를 기대합니다",
		"log.unsupported_method":               "지원되지 않는 요청 메서드",
		"log.missing_query_params":             "쿼리 매개변수 누락, phone, mail 또는 user_id 제공 필요",
		"log.multiple_query_params":            "하나의 쿼리 매개변수만 허용됨 (phone, mail 또는 user_id)",
		"log.user_not_found":                   "사용자를 찾을 수 없습니다",
		"log.user_query_success":               "사용자 쿼리 성공",
		"log.pagination_validation_failed":     "페이지 매김 매개변수 유효성 검사 실패",
		"log.request_data_api":                 "데이터 API 요청",
		"log.health_check_encode_failed":       "상태 확인 응답 인코딩 실패",

		// HTTP response messages
		"http.method_not_allowed":            "Method not allowed",
		"http.user_not_found":                "User not found",
		"http.internal_server_error":         "Internal server error",
		"http.unauthorized":                  "Unauthorized",
		"http.rate_limit_exceeded":           "Rate limit exceeded",
		"http.invalid_pagination_parameters": "Invalid pagination parameters",
	},
}

// T returns the translated string for the given key from request context
// If the key is not found, it returns the key itself
func T(r *http.Request, key string) string {
	lang := GetLanguageFromContext(r)
	return translate(lang, key)
}

// Tf returns a formatted translated string from request context
func Tf(r *http.Request, key string, args ...interface{}) string {
	return fmt.Sprintf(T(r, key), args...)
}

// TWithLang returns the translated string for the given key with specified language
func TWithLang(lang Language, key string) string {
	return translate(lang, key)
}

// TfWithLang returns a formatted translated string with specified language
func TfWithLang(lang Language, key string, args ...interface{}) string {
	return fmt.Sprintf(TWithLang(lang, key), args...)
}

// translate 内部翻译函数
func translate(lang Language, key string) string {
	if langMap, ok := translations[lang]; ok {
		if translation, ok := langMap[key]; ok {
			return translation
		}
	}

	// Fallback to English if translation not found
	if langMap, ok := translations[LangEN]; ok {
		if translation, ok := langMap[key]; ok {
			return translation
		}
	}

	// Return key if no translation found
	return key
}
