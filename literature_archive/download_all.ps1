$ErrorActionPreference = 'Stop'

$archiveRoot = Split-Path -Parent $PSScriptRoot
$papersDir = Join-Path $PSScriptRoot 'papers'
New-Item -ItemType Directory -Force -Path $papersDir | Out-Null

$papers = @(
    @{ Id = 'P01'; File = 'P01_guo_2024_llm_multi_agents_survey.pdf'; Url = 'https://arxiv.org/pdf/2402.01680' },
    @{ Id = 'P02'; File = 'P02_slms_future_agentic_ai_2025.pdf'; Url = 'https://arxiv.org/pdf/2506.02153' },
    @{ Id = 'P03'; File = 'P03_llm_slm_collaboration_survey_2025.pdf'; Url = 'https://arxiv.org/pdf/2505.07460' },
    @{ Id = 'P04'; File = 'P04_marco_2024.pdf'; Url = 'https://arxiv.org/pdf/2410.21784' },
    @{ Id = 'P05'; File = 'P05_fellowship_llms_2024.pdf'; Url = 'https://arxiv.org/pdf/2408.08688' },
    @{ Id = 'P06'; File = 'P06_scaling_small_agents_strategy_auctions_2026.pdf'; Url = 'https://arxiv.org/pdf/2602.02751' },
    @{ Id = 'P07'; File = 'P07_rethinking_value_multi_agent_workflow_2026.pdf'; Url = 'https://arxiv.org/pdf/2601.12307' },
    @{ Id = 'P08'; File = 'P08_rethinking_scale_slm_agent_paradigms_2026.pdf'; Url = 'https://arxiv.org/pdf/2604.19299' },
    @{ Id = 'P09'; File = 'P09_auto_slurp_2025.pdf'; Url = 'https://aclanthology.org/2025.findings-emnlp.596.pdf' },
    @{ Id = 'P10'; File = 'P10_riskagent_2025.pdf'; Url = 'https://arxiv.org/pdf/2503.03802' },
    @{ Id = 'P11'; File = 'P11_llmsr_xllm_2025.pdf'; Url = 'https://aclanthology.org/2025.xllm-1.23.pdf' },
    @{ Id = 'P12'; File = 'P12_can_small_agents_collaborate_2026.pdf'; Url = 'https://arxiv.org/pdf/2601.11327' },
    @{ Id = 'P13'; File = 'P13_llm_mas_applications_survey_2024.pdf'; Url = 'https://arxiv.org/pdf/2412.17481' },
    @{ Id = 'P14'; File = 'P14_planner_matters_2026.pdf'; Url = 'https://arxiv.org/pdf/2605.02168' },
    @{ Id = 'P15'; File = 'P15_merit_multi_agent_time_series_2025.pdf'; Url = 'https://aclanthology.org/2025.findings-acl.1231.pdf' },
    @{ Id = 'P16'; File = 'P16_beyond_monolithic_agentic_search_2026.pdf'; Url = 'https://arxiv.org/pdf/2601.04703' },
    @{ Id = 'P17'; File = 'P17_multi_agent_collaboration_entropy_2026.pdf'; Url = 'https://arxiv.org/pdf/2602.04234' },
    @{ Id = 'P18'; File = 'P18_consensus_multi_agent_2025.pdf'; Url = 'https://aclanthology.org/2025.emnlp-industry.144.pdf' },
    @{ Id = 'P19'; File = 'P19_orchestrator_clinical_support_2025.pdf'; Url = 'https://arxiv.org/pdf/2512.04207' },
    @{ Id = 'P20'; File = 'P20_agentleak_2026.pdf'; Url = 'https://arxiv.org/pdf/2602.11510' },
    @{ Id = 'P21'; File = 'P21_consensagent_2025.pdf'; Url = 'https://aclanthology.org/2025.findings-acl.1141.pdf' },
    @{ Id = 'P22'; File = 'P22_maporl2_2025.pdf'; Url = 'https://aclanthology.org/2025.acl-long.1459.pdf' },
    @{ Id = 'P23'; File = 'P23_score_student_centered_distillation_2025.pdf'; Url = 'https://arxiv.org/pdf/2509.14257' },
    @{ Id = 'P24'; File = 'P24_mad_opd_2026.pdf'; Url = 'https://arxiv.org/pdf/2605.01347' },
    @{ Id = 'P25'; File = 'P25_chain_of_agents_2025.pdf'; Url = 'https://arxiv.org/pdf/2508.13167' },
    @{ Id = 'P26'; File = 'P26_agentdistill_2025.pdf'; Url = 'https://arxiv.org/pdf/2506.14728' },
    @{ Id = 'P27'; File = 'P27_distilling_llm_agent_small_models_2025.pdf'; Url = 'https://arxiv.org/pdf/2505.17612' },
    @{ Id = 'P28'; File = 'P28_r2v_agent_2026.pdf'; Url = 'https://arxiv.org/pdf/2605.16604' }
)

$result = foreach ($paper in $papers) {
    $destination = Join-Path $papersDir $paper.File
    try {
        $existing = Test-Path -LiteralPath $destination
        if (-not $existing -or (Get-Item -LiteralPath $destination).Length -lt 1024) {
            curl.exe --fail --location --retry 3 --retry-delay 2 --connect-timeout 20 --max-time 90 --output $destination $paper.Url
            if (-not (Test-Path -LiteralPath $destination) -or (Get-Item -LiteralPath $destination).Length -lt 1024) {
                throw 'Downloaded file is missing or unexpectedly small.'
            }
        }
        $hash = (Get-FileHash -LiteralPath $destination -Algorithm SHA256).Hash
        [PSCustomObject]@{
            id = $paper.Id
            file = $paper.File
            url = $paper.Url
            status = 'downloaded'
            size_bytes = (Get-Item -LiteralPath $destination).Length
            sha256 = $hash
        }
    }
    catch {
        if (Test-Path -LiteralPath $destination) { Remove-Item -LiteralPath $destination -Force }
        [PSCustomObject]@{
            id = $paper.Id
            file = $paper.File
            url = $paper.Url
            status = "failed: $($_.Exception.Message)"
            size_bytes = 0
            sha256 = ''
        }
    }
}

$result | ConvertTo-Json -Depth 3 | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'download_manifest.json') -Encoding utf8
$result | Export-Csv -LiteralPath (Join-Path $PSScriptRoot 'download_manifest.csv') -NoTypeInformation -Encoding utf8
$result | Format-Table -AutoSize
