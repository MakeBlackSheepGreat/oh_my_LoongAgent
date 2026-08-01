$ErrorActionPreference = 'Stop'

$work = Join-Path $env:TEMP 'literature_comparison_update_20260730'
$sheet1Path = Join-Path $work 'xl\worksheets\sheet1.xml'
$sheet2Path = Join-Path $work 'xl\worksheets\sheet2.xml'
$output = Join-Path $PSScriptRoot 'literature_comparison.xlsx'
$packer = 'C:\Users\876762330\.codex\skills\minimax-xlsx\scripts\xlsx_pack.py'

function Escape-Xml([string]$Value) {
    [System.Security.SecurityElement]::Escape($Value)
}

function Column-Name([int]$Number) {
    $name = ''
    while ($Number -gt 0) {
        $remainder = ($Number - 1) % 26
        $name = [char](65 + $remainder) + $name
        $Number = [math]::Floor(($Number - 1) / 26)
    }
    $name
}

function Inline-Cell([string]$Address, [string]$Value) {
    '<c r="{0}" t="inlineStr" s="13"><is><t xml:space="preserve">{1}</t></is></c>' -f $Address, (Escape-Xml $Value)
}

if (-not (Test-Path -LiteralPath $sheet1Path) -or -not (Test-Path -LiteralPath $sheet2Path)) {
    throw 'Expected unpacked workbook XML is missing.'
}

$records = @(
    @('P23','Student-Centered Distillation Narrows the Agentic Gap Between Small and Large LLMs','2025 / arXiv','7B 学生；72B 教师','学生轨迹 + 最早错误点修正 + 短程 RL','12 个挑战性 Agent 基准','7B; 蒸馏; 教师','最接近强教师对小 Agent 的闭环','P23_score_student_centered_distillation_2025.pdf'),
    @('P24','MAD-OPD: Breaking the Ceiling in On-Policy Distillation via Multi-Agent Debate','2026 / arXiv','1.7B-14B 学生；8B-32B 教师','多教师辩论 + 在线策略蒸馏','5 个 Agent 与代码基准','8B; 蒸馏; 辩论','多教师监督与 8B 教师参考','P24_mad_opd_2026.pdf'),
    @('P25','Chain-of-Agents: End-to-End Agent Foundation Models via Multi-Agent Distillation and Agentic RL','2025 / arXiv','多 Agent 系统蒸馏到 AFM','协作轨迹蒸馏 + Agentic RL','Web Agent 与 Code Agent','蒸馏; 多Agent; 工具','将现有 MAS 能力压缩到可训练模型','P25_chain_of_agents_2025.pdf'),
    @('P26','AgentDistill: Training-Free Agent Distillation with Generalizable MCP Boxes','2025 / arXiv','小语言模型学生；GPT-4o 系统对照','复用教师生成的结构化 MCP 模块','生物医学与数学基准','蒸馏; MCP; 工具','训练外蒸馏和可复用技能参考','P26_agentdistill_2025.pdf'),
    @('P27','Distilling LLM Agent into Small Models with Retrieval and Code Tools','2025 / NeurIPS Spotlight','0.5B、1.5B、3B 学生','教师轨迹、检索/代码工具与自一致动作','8 个推理任务','蒸馏; 工具; 检索','工具轨迹蒸馏的直接对照','P27_distilling_llm_agent_small_models_2025.pdf'),
    @('P28','R2V Agent: Teaching SLMs When to Ask for Help','2026 / arXiv','4 个 SLM 骨干 + 强教师','逐步风险估计与条件升级','HumanEval+、TextWorld、TerminalBench','路由; 教师; 工具','只在高风险步骤调用强模型的参考','P28_r2v_agent_2026.pdf')
)

$sheet1 = Get-Content -LiteralPath $sheet1Path -Raw
if ($sheet1 -match 'P23') { throw 'P23 already exists in the workbook.' }
$newRows = foreach ($i in 0..($records.Count - 1)) {
    $rowNumber = 24 + $i
    $cells = foreach ($j in 0..8) {
        Inline-Cell ((Column-Name ($j + 1)) + $rowNumber) $records[$i][$j]
    }
    '<row r="{0}" ht="56" customHeight="1">{1}</row>' -f $rowNumber, ($cells -join '')
}
$sheet1 = $sheet1.Replace('</sheetData>', (($newRows -join '') + '</sheetData>'))
$sheet1 = $sheet1.Replace('autoFilter ref="A1:I23"', 'autoFilter ref="A1:I29"')
Set-Content -LiteralPath $sheet1Path -Value $sheet1 -Encoding utf8

$sheet2 = Get-Content -LiteralPath $sheet2Path -Raw
$sheet2 = $sheet2.Replace("'文献对比'!A2:A23", "'文献对比'!A2:A29")
$sheet2 = $sheet2.Replace("'文献对比'!G2:G23", "'文献对比'!G2:G29")
Set-Content -LiteralPath $sheet2Path -Value $sheet2 -Encoding utf8

& python $packer $work $output
if ($LASTEXITCODE -ne 0) { throw 'XLSX packing failed.' }
Write-Output $output
