package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

type Item struct {
	Country     string  `json:"country"`
	Name        string  `json:"name"`
	Desc        string  `json:"desc"`
	Roast       string  `json:"roast"`
	Emoji       string  `json:"emoji"`
	SearchQuery string  `json:"searchQuery"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
}

type LLMResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

var countryMap = map[string][2]float64{
	// 亚洲
	"China":        {35.8617, 104.1954},
	"ChinaTaiwan":  {23.6978, 120.9605},
	"Taiwan":       {23.6978, 120.9605},
	"ChinaHongKong":{22.3193, 114.1694},
	"HongKong":     {22.3193, 114.1694},
	"ChinaMacau":   {22.1987, 113.5439},
	"Macau":        {22.1987, 113.5439},
	"India":       {20.5937, 78.9629},
	"Japan":       {36.2048, 138.2529},
	"SouthKorea":  {35.9078, 127.7669},
	"Thailand":    {15.8700, 100.9925},
	"Vietnam":     {14.0583, 108.2772},
	"Indonesia":   {-0.7893, 113.9213},
	"Turkey":      {38.9637, 35.2433},
	"SaudiArabia": {23.8859, 45.0792},
	"Iran":        {32.4279, 53.6880},
	// 欧洲
	"France":         {46.2276, 2.2137},
	"Sweden":         {60.1282, 18.6435},
	"Italy":          {41.8719, 12.5674},
	"Spain":          {40.4637, -3.7492},
	"Germany":        {51.1657, 10.4515},
	"Greece":         {39.0742, 21.8243},
	"Portugal":       {39.3999, -8.2245},
	"Russia":         {61.5240, 105.3188},
	"Poland":         {51.9194, 19.1451},
	"UnitedKingdom":  {55.3781, -3.4360},
	// 美洲
	"UnitedStates": {37.0902, -95.7129},
	"Brazil":       {-14.2350, -51.9253},
	"Mexico":       {23.6345, -102.5528},
	"Argentina":    {-38.4161, -63.6167},
	"Peru":         {-9.1900, -75.0152},
	"Colombia":     {4.5709, -74.2973},
	"Canada":       {56.1304, -106.3468},
	"Chile":        {-35.6751, -71.5430},
	// 非洲
	"Morocco":      {31.7917, -7.0926},
	"Ethiopia":     {9.1450, 40.4897},
	"Nigeria":      {9.0820, 8.6753},
	"SouthAfrica":  {-30.5595, 22.9375},
	"Egypt":        {26.8206, 30.8025},
	// 大洋洲
	"Australia":   {-25.2744, 133.7751},
	"NewZealand":  {-40.9006, 174.8860},
	// 大洲兜底
	"Asia":         {34.0479, 100.6197},
	"Europe":       {54.5260, 15.2551},
	"Africa":       {8.7832, 34.5085},
	"NorthAmerica": {54.5260, -105.2551},
	"SouthAmerica": {-8.7832, -55.4915},
	"Oceania":      {-22.7359, 140.0188},
	"Antarctica":   {-82.8628, 135.0000},
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/api/generate", handler)
	http.HandleFunc("/api/publish", publishHandler)
	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"siteUrl": os.Getenv("SITE_URL"),
		})
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	fmt.Println("Server running on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var req struct {
		Query string `json:"query"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	items, err := callLLM(req.Query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// 记录每个国家出现次数，给重叠的点加偏移
	countryCount := map[string]int{}
	for i := range items {
		key := strings.ReplaceAll(items[i].Country, " ", "")
		if coord, ok := countryMap[key]; ok {
			n := countryCount[key]
			countryCount[key]++
			angle := float64(n) * 2.5
			offset := 1.5
			items[i].Lat = coord[0] + offset*math.Sin(angle)
			items[i].Lng = coord[1] + offset*math.Cos(angle)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func callLLM(query string) ([]Item, error) {
	prompt := fmt.Sprintf(`仅返回 JSON 数组，不要任何解释或 markdown。

主题：%s

规则：
- 每个大洲返回 3-5 个与主题**严格相关**的事物（尽量覆盖全部 7 大洲，总数不少于 20 个）
- 每个条目必须真正属于主题类别，不能偏题。例如主题是"动物"就只返回动物，主题是"食物"就只返回食物
- "country" 必须是真实国家的英文名且不含空格（如 "UnitedStates"、"SouthKorea"），绝对不能用大洲名；台湾、香港、澳门分别用 "ChinaTaiwan"、"ChinaHongKong"、"ChinaMacau"
- "name" 用中文名称
- "emoji" 必须是一个 Unicode emoji 表情符号（如 🐊、🐍、🦁），不能用中文文字
- "desc" 恰好两行，用 \n 分隔：第一行是所在地/来源，第二行是简短的类别或特征描述
- "roast" 是一句幽默吐槽（中文）
- "searchQuery" 是一个适合搜索该事物的中文短语，格式如"主题+名称"，例如"世界上最奇葩的厕所：美国波士顿监狱厕所"

示例格式：
[
  {
    "country": "Japan",
    "name": "新干线厕所",
    "emoji": "🚽",
    "desc": "所在地：日本新干线列车\n特征：全自动智能马桶，带音乐遮音功能",
    "roast": "上个厕所需要先读说明书。",
    "searchQuery": "世界上最奇葩的厕所：日本新干线智能马桶"
  }
]
`, query)

	reqBody := map[string]interface{}{
		"model":      "MiniMax-Text-01",
		"max_tokens": 4096,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "https://api.minimax.chat/v1/text/chatcompletion_v2", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+os.Getenv("MINIMAX_API_KEY"))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("LLM raw response: %s", string(respBody))

	var llmResp LLMResponse
	json.Unmarshal(respBody, &llmResp)

	if len(llmResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	text := llmResp.Choices[0].Message.Content

	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start == -1 || end == -1 {
		return nil, fmt.Errorf("invalid JSON")
	}

	var items []Item
	err = json.Unmarshal([]byte(text[start:end+1]), &items)
	return items, err
}

func publishHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var req struct {
		Name        string `json:"name"`
		SearchQuery string `json:"searchQuery"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	title := req.SearchQuery
	if !strings.HasSuffix(title, "？") {
		title += "？"
	}
	desc := fmt.Sprintf("关于 %s 的疑问", req.Name)

	zhihuCookie := os.Getenv("ZHIHU_COOKIE")
	if zhihuCookie == "" {
		http.Error(w, "未配置知乎 Cookie", 500)
		return
	}

	payload := map[string]interface{}{
		"action": "question",
		"data": map[string]interface{}{
			"publish": map[string]string{
				"traceId": fmt.Sprintf("%d,603f9ae3-8db4-4f9a-9e1f-2fbfced5aadb", time.Now().UnixMilli()),
			},
			"title": map[string]string{"title": title},
			"topic": map[string][]string{"topics": {}},
			"hybrid": map[string]interface{}{
				"html":       fmt.Sprintf("<p>%s</p>", desc),
				"textLength": len(desc),
			},
			"extra_info":     map[string]string{"publisher": "pc"},
			"questionConfig": map[string]string{"brand_id": "undefined", "type": "0"},
		},
	}

	body, _ := json.Marshal(payload)
	zhReq, _ := http.NewRequest("POST", "https://www.zhihu.com/api/v4/content/publish", bytes.NewBuffer(body))
	zhReq.Header.Set("Content-Type", "application/json")
	zhReq.Header.Set("Cookie", zhihuCookie)

	client := &http.Client{}
	resp, err := client.Do(zhReq)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}
