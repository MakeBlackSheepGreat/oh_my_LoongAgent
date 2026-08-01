$ErrorActionPreference = 'Stop'

$archiveRoot = $PSScriptRoot
$outputFile = Join-Path $archiveRoot 'literature_comparison.xlsx'
$template = 'C:\Users\876762330\.codex\skills\minimax-xlsx\templates\minimal_xlsx'
$packer = 'C:\Users\876762330\.codex\skills\minimax-xlsx\scripts\xlsx_pack.py'
$work = Join-Path $env:TEMP ('literature_xlsx_' + [Guid]::NewGuid().ToString('N'))

function Escape-Xml([string]$Value) {
    return [System.Security.SecurityElement]::Escape($Value)
}

function Column-Name([int]$Number) {
    $name = ''
    while ($Number -gt 0) {
        $remainder = ($Number - 1) % 26
        $name = [char](65 + $remainder) + $name
        $Number = [math]::Floor(($Number - 1) / 26)
    }
    return $name
}

function Inline-Cell([string]$Address, [string]$Value, [int]$Style = 13) {
    return ('<c r="{0}" t="inlineStr" s="{1}"><is><t xml:space="preserve">{2}</t></is></c>' -f $Address, $Style, (Escape-Xml $Value))
}

function Formula-Cell([string]$Address, [string]$Formula) {
    return ('<c r="{0}" s="2"><f>{1}</f><v></v></c>' -f $Address, (Escape-Xml $Formula))
}

$records = @(
    @('P01','Large Language Model based Multi-Agents: A Survey of Progress and Challenges','2024 / arXiv','综述','MAS 架构、协作与挑战','多领域','综述','理论与术语基线','P01_guo_2024_llm_multi_agents_survey.pdf'),
    @('P02','Small Language Models are the Future of Agentic AI','2025 / arXiv','SLM，10B 以下讨论','SLM 与大模型协作、替换','Agent 系统部署','SLM; 路由','小模型基础执行层论证','P02_slms_future_agentic_ai_2025.pdf'),
    @('P03','A Survey on Collaborative Mechanisms Between Large and Small Language Models','2025 / arXiv','LLM 与 SLM','路由、辅助、蒸馏、融合','协作机制综述','SLM; 路由','大小模型协同综述','P03_llm_slm_collaboration_survey_2025.pdf'),
    @('P04','MARCO: Multi-Agent Real-time Chat Orchestration','2024 / EMNLP Industry','Llama-3-8B-Instruct；Mistral-7B','实时编排、工具与错误恢复','零售与餐饮对话','8B; 工具; 时延','8B 任务编排先例','P04_marco_2024.pdf'),
    @('P05','The Fellowship of the LLMs: Multi-Model Workflows for Synthetic Preference Optimization Dataset Generation','2024 / arXiv','Llama-3.1-8B；Gemma-2-9B','生成者与审查者反馈闭环','偏好数据生成','8B; 审查','执行与审查角色分工','P05_fellowship_llms_2024.pdf'),
    @('P06','Scaling Small Agents Through Strategy Auctions','2026 / ICML','小 Agent','策略拍卖式任务分配','复杂任务与成本','小模型; 路由','按能力分工的机制参考','P06_scaling_small_agents_strategy_auctions_2026.pdf'),
    @('P07','Rethinking the Value of Multi-Agent Workflow: A Strong Single Agent Baseline','2026 / arXiv','多模型配置','同质 MAS 与强单 Agent 对照','7 个基准','单Agent; 基线','必须复现实验对照','P07_rethinking_value_multi_agent_workflow_2026.pdf'),
    @('P08','Rethinking Scale: Deployment Trade-offs of Small Language Models under Agent Paradigms','2026 / ACL Industry','27 个开源 SLM，均小于 10B','基础、工具单 Agent、协作 MAS','20 金融数据集，8 类任务','小模型; 基线; 效率','实验设计最接近','P08_rethinking_scale_slm_agent_paradigms_2026.pdf'),
    @('P09','Auto-SLURP: A Benchmark for Evaluating Multi-Agent Frameworks in Smart Personal Assistant','2025 / Findings EMNLP','Llama-3 8B 微调意图模块；其余为 GPT-4','管理者协调理解、服务调用与响应','智能个人助手','8B; 日常; 基准','日常任务与基准设计先例','P09_auto_slurp_2025.pdf'),
    @('P10','RiskAgent: Synergizing Language Models with Validated Tools for Evidence-Based Risk Prediction','2025 / arXiv','RiskAgent-8B，LLaMA-3-8B','决策、执行、审查与工具协同','医学风险预测','8B; 工具; 审查','端到端 8B 工具系统','P10_riskagent_2025.pdf'),
    @('P11','LLMSR@XLLM25: Less is More: Enhancing Structured Multi-Agent Reasoning via Quality-Guided Distillation','2025 / XLLM Workshop','Meta-Llama-3-8B-Instruct + LoRA','多角色推理、检索与质量过滤','低资源结构化推理','8B; 蒸馏','全流程 8B 多 Agent 案例','P11_llmsr_xllm_2025.pdf'),
    @('P12','Can Small Agents Collaborate to Beat a Single Large Language Model?','2026 / ICLR MALGAI Workshop','小模型协作','受控协作策略','工具密集型基准','小模型; 基线','多 Agent 适用条件参考','P12_can_small_agents_collaborate_2026.pdf'),
    @('P13','A Survey on LLM-based Multi-Agent System: Recent Advances and New Frontiers in Application','2024 / arXiv','综述','工作流、基础设施与应用','多领域','综述','应用与工程背景','P13_llm_mas_applications_survey_2024.pdf'),
    @('P14','Planner Matters! An Efficient and Unbalanced Multi-agent Collaboration Framework for Long-horizon Planning','2026 / arXiv','Qwen2.5-VL-7B','规划者主导的非均衡协作','长程 GUI 任务','7B; 日常; GUI','小模型长程规划参考','P14_planner_matters_2026.pdf'),
    @('P15','MERIT: Multi-Agent Collaboration for Unsupervised Time Series Representation Learning','2025 / Findings ACL','多模型协作','时间序列表征的协作学习','时间序列任务','多Agent','跨任务协作方法参考','P15_merit_multi_agent_time_series_2025.pdf'),
    @('P16','Beyond Monolithic Architectures: A Multi-Agent Search and Knowledge Optimization Framework for Agentic Search','2026 / arXiv','Qwen2.5-7B','检索、知识优化与多 Agent 搜索','搜索与知识密集任务','7B; 搜索; 工具','RAG 与搜索编排参考','P16_beyond_monolithic_agentic_search_2026.pdf'),
    @('P17','When Does Multi-Agent Collaboration Help? An Entropy Perspective','2026 / arXiv','Qwen3-8B','基于任务不确定性的协作触发','工具使用与复杂任务','8B; 路由; 基线','何时启用多 Agent 的核心参考','P17_multi_agent_collaboration_entropy_2026.pdf'),
    @('P18','AMAS: Adaptively Determining Communication Topology for LLM-based Multi-agent System','2025 / EMNLP Industry','多模型配置','自适应通信拓扑','多 Agent 协作任务','拓扑; 效率','通信拓扑与成本控制参考','P18_consensus_multi_agent_2025.pdf'),
    @('P19','Orchestrator Multi-Agent Clinical Decision Support System for Secondary Headache Diagnosis in Primary Care','2025 / arXiv','Llama 3.1 8B 等开源模型','临床工作流角色协同','基层医疗决策支持','8B; 工具; 工作流','垂直领域端到端部署参考','P19_orchestrator_clinical_support_2025.pdf'),
    @('P20','AgentLeak: A Benchmark for Internal-Channel Privacy Leakage in Multi-Agent LLM Systems','2026 / IEEE Access','多模型配置','多 Agent 内部通信安全','隐私泄露基准','安全; 基准','安全评测指标参考','P20_agentleak_2026.pdf'),
    @('P21','CONSENSAGENT: Towards Efficient and Effective Consensus in Multi-Agent LLM Interactions through Sycophancy Mitigation','2025 / Findings ACL','Llama 系列模型','共识与奉承抑制','多 Agent 交互评测','共识; 可靠性','协作可靠性与群体偏差参考','P21_consensagent_2025.pdf'),
    @('P22','MAPoRL2: Multi-Agent Post-Co-Training for Collaborative Large Language Models with Reinforcement Learning','2025 / ACL','3B 与 8B 模型','同伴驱动强化学习训练','通用推理任务','8B; 训练','8B Agent 训练策略参考','P22_maporl2_2025.pdf'),
    @('P23','Student-Centered Distillation Narrows the Agentic Gap Between Small and Large LLMs','2025 / arXiv','7B 学生；72B 教师','学生轨迹 + 最早错误点修正 + 短程 RL','12 个挑战性 Agent 基准','7B; 蒸馏; 教师','最接近强教师对小 Agent 的闭环','P23_score_student_centered_distillation_2025.pdf'),
    @('P24','MAD-OPD: Breaking the Ceiling in On-Policy Distillation via Multi-Agent Debate','2026 / arXiv','1.7B-14B 学生；8B-32B 教师','多教师辩论 + 在线策略蒸馏','5 个 Agent 与代码基准','8B; 蒸馏; 辩论','多教师监督与 8B 教师参考','P24_mad_opd_2026.pdf'),
    @('P25','Chain-of-Agents: End-to-End Agent Foundation Models via Multi-Agent Distillation and Agentic RL','2025 / arXiv','多 Agent 系统蒸馏到 AFM','协作轨迹蒸馏 + Agentic RL','Web Agent 与 Code Agent','蒸馏; 多Agent; 工具','将现有 MAS 能力压缩到可训练模型','P25_chain_of_agents_2025.pdf'),
    @('P26','AgentDistill: Training-Free Agent Distillation with Generalizable MCP Boxes','2025 / arXiv','小语言模型学生；GPT-4o 系统对照','复用教师生成的结构化 MCP 模块','生物医学与数学基准','蒸馏; MCP; 工具','训练外蒸馏和可复用技能参考','P26_agentdistill_2025.pdf'),
    @('P27','Distilling LLM Agent into Small Models with Retrieval and Code Tools','2025 / NeurIPS Spotlight','0.5B、1.5B、3B 学生','教师轨迹、检索/代码工具与自一致动作','8 个推理任务','蒸馏; 工具; 检索','工具轨迹蒸馏的直接对照','P27_distilling_llm_agent_small_models_2025.pdf'),
    @('P28','R2V Agent: Teaching SLMs When to Ask for Help','2026 / arXiv','4 个 SLM 骨干 + 强教师','逐步风险估计与条件升级','HumanEval+、TextWorld、TerminalBench','路由; 教师; 工具','只在高风险步骤调用强模型的参考','P28_r2v_agent_2026.pdf')
)

Copy-Item -LiteralPath $template -Destination $work -Recurse

$stylesPath = Join-Path $work 'xl\styles.xml'
$styles = Get-Content -LiteralPath $stylesPath -Raw
$styles = $styles.Replace('<cellXfs count="13">', '<cellXfs count="14">')
$styles = $styles.Replace('</cellXfs>', '<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0" applyAlignment="1"><alignment vertical="top" wrapText="1"/></xf></cellXfs>')
Set-Content -LiteralPath $stylesPath -Value $styles -Encoding utf8

$headers = @('ID','论文','年份 / 载体','模型规模或骨干','Agent 机制','任务或评测','标签','与项目关系','归档 PDF')
$widths = @(8,65,24,35,45,34,24,40,48)
$sheetRows = @()
$headerCells = for ($i = 0; $i -lt $headers.Count; $i++) { Inline-Cell ((Column-Name ($i + 1)) + '1') $headers[$i] 4 }
$sheetRows += '<row r="1" ht="30" customHeight="1">' + ($headerCells -join '') + '</row>'
for ($rowIndex = 0; $rowIndex -lt $records.Count; $rowIndex++) {
    $cells = for ($colIndex = 0; $colIndex -lt $headers.Count; $colIndex++) {
        Inline-Cell ((Column-Name ($colIndex + 1)) + ($rowIndex + 2)) $records[$rowIndex][$colIndex]
    }
    $sheetRows += '<row r="' + ($rowIndex + 2) + '" ht="56" customHeight="1">' + ($cells -join '') + '</row>'
}
$cols = for ($i = 0; $i -lt $widths.Count; $i++) { '<col min="' + ($i + 1) + '" max="' + ($i + 1) + '" width="' + $widths[$i] + '" customWidth="1"/>' }
$sheet1 = @"
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>
  <sheetFormatPr defaultRowHeight="15"/>
  <cols>$($cols -join '')</cols>
  <sheetData>$($sheetRows -join '')</sheetData>
  <autoFilter ref="A1:I$($records.Count + 1)"/>
  <pageMargins left="0.4" right="0.4" top="0.6" bottom="0.6" header="0.3" footer="0.3"/>
</worksheet>
"@
Set-Content -LiteralPath (Join-Path $work 'xl\worksheets\sheet1.xml') -Value $sheet1 -Encoding utf8

$summaryRows = @(
    '<row r="1" ht="30" customHeight="1">' + (Inline-Cell 'A1' '检索与归档摘要' 4) + '</row>',
    '<row r="3">' + (Inline-Cell 'A3' '统计项' 4) + (Inline-Cell 'B3' '数值' 4) + '</row>',
    '<row r="4">' + (Inline-Cell 'A4' '纳入文献数') + (Formula-Cell 'B4' "COUNTA('文献对比'!A2:A$($records.Count + 1))") + '</row>',
    '<row r="5">' + (Inline-Cell 'A5' '直接涉及 8B 的文献数') + (Formula-Cell 'B5' "COUNTIF('文献对比'!G2:G$($records.Count + 1),\"*8B*\")") + '</row>',
    '<row r="6">' + (Inline-Cell 'A6' '直接涉及日常或个人助手任务的文献数') + (Formula-Cell 'B6' "COUNTIF('文献对比'!G2:G$($records.Count + 1),\"*日常*\")") + '</row>',
    '<row r="7">' + (Inline-Cell 'A7' '文献归档状态') + (Inline-Cell 'B7' 'download_manifest.json 记录每份 PDF 的 SHA-256') + '</row>',
    '<row r="9">' + (Inline-Cell 'A9' '检索范围' 4) + '</row>',
    '<row r="10">' + (Inline-Cell 'A10' '时间') + (Inline-Cell 'B10' '2026-07-30') + '</row>',
    '<row r="11">' + (Inline-Cell 'A11' '来源') + (Inline-Cell 'B11' 'arXiv、ACL Anthology、ICML、IEEE Access') + '</row>',
    '<row r="12">' + (Inline-Cell 'A12' '筛选条件') + (Inline-Cell 'B12' '多 Agent、小模型或 7B/8B、工具调用、日常任务、基准、路由与可靠性') + '</row>'
)
$sheet2 = @"
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheetViews><sheetView workbookViewId="0"><pane ySplit="3" topLeftCell="A4" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>
  <sheetFormatPr defaultRowHeight="18"/>
  <cols><col min="1" max="1" width="36" customWidth="1"/><col min="2" max="2" width="85" customWidth="1"/></cols>
  <sheetData>$($summaryRows -join '')</sheetData>
  <pageMargins left="0.4" right="0.4" top="0.6" bottom="0.6" header="0.3" footer="0.3"/>
</worksheet>
"@
Set-Content -LiteralPath (Join-Path $work 'xl\worksheets\sheet2.xml') -Value $sheet2 -Encoding utf8

$contentTypes = @"
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/worksheets/sheet2.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
  <Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
</Types>
"@
Set-Content -LiteralPath (Join-Path $work '[Content_Types].xml') -Value $contentTypes -Encoding utf8

$workbook = @"
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <bookViews><workbookView/></bookViews>
  <sheets><sheet name="文献对比" sheetId="1" r:id="rId1"/><sheet name="检索摘要" sheetId="2" r:id="rId4"/></sheets>
  <calcPr calcId="191029" fullCalcOnLoad="1" forceFullCalc="1"/>
</workbook>
"@
Set-Content -LiteralPath (Join-Path $work 'xl\workbook.xml') -Value $workbook -Encoding utf8

$rels = @"
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>
  <Relationship Id="rId4" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/>
</Relationships>
"@
Set-Content -LiteralPath (Join-Path $work 'xl\_rels\workbook.xml.rels') -Value $rels -Encoding utf8

& python $packer $work $outputFile
if ($LASTEXITCODE -ne 0) { throw 'XLSX packing failed.' }
Remove-Item -LiteralPath $work -Recurse -Force
Write-Output $outputFile
