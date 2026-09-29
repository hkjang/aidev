param(
    [int]$Size = 256,
    [string]$OutPath = "assets\icons\icon.png"
)

Add-Type -AssemblyName System.Drawing

$root = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$out = Join-Path $root $OutPath

$bm = New-Object System.Drawing.Bitmap $Size, $Size
$g = [System.Drawing.Graphics]::FromImage($bm)
$g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
$g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
$g.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAlias

# === 1. 어두운 보라색 둥근 사각형 배경 ===
$margin = [int]($Size * 0.025)
$corner = [int]($Size * 0.16)
$bgRect = New-Object System.Drawing.Rectangle $margin, $margin, ($Size - 2 * $margin), ($Size - 2 * $margin)
$path = New-Object System.Drawing.Drawing2D.GraphicsPath
$d = $corner * 2
$path.AddArc($bgRect.X, $bgRect.Y, $d, $d, 180, 90)
$path.AddArc($bgRect.Right - $d, $bgRect.Y, $d, $d, 270, 90)
$path.AddArc($bgRect.Right - $d, $bgRect.Bottom - $d, $d, $d, 0, 90)
$path.AddArc($bgRect.X, $bgRect.Bottom - $d, $d, $d, 90, 90)
$path.CloseFigure()

$bgGrad = New-Object System.Drawing.Drawing2D.LinearGradientBrush(
    (New-Object System.Drawing.Point 0, 0),
    (New-Object System.Drawing.Point $Size, $Size),
    [System.Drawing.Color]::FromArgb(255, 22, 18, 38),
    [System.Drawing.Color]::FromArgb(255, 40, 30, 70)
)
$g.FillPath($bgGrad, $path)

# 외곽 라이트 보더
$borderPen = New-Object System.Drawing.Pen ([System.Drawing.Color]::FromArgb(180, 150, 130, 220)), 2.0
$g.DrawPath($borderPen, $path)

# === 2. V 모양 (왼쪽 보라, 오른쪽 청) ===
$thickness = [int]($Size * 0.11)

# V 좌표 (256 기준)
# 왼쪽 사선: (60, 88) → (128, 218)
# 오른쪽 사선: (128, 218) → (196, 88)
$s = $Size / 256.0

$leftTopX = 60 * $s; $leftTopY = 88 * $s
$bottomX = 128 * $s; $bottomY = 218 * $s
$rightTopX = 196 * $s; $rightTopY = 88 * $s

# 왼쪽 보라 사선
$purplePen = New-Object System.Drawing.Pen ([System.Drawing.Color]::FromArgb(255, 142, 82, 230)), $thickness
$purplePen.StartCap = [System.Drawing.Drawing2D.LineCap]::Round
$purplePen.EndCap = [System.Drawing.Drawing2D.LineCap]::Round
$g.DrawLine($purplePen, $leftTopX, $leftTopY, $bottomX, $bottomY)

# 오른쪽 청색 사선
$bluePen = New-Object System.Drawing.Pen ([System.Drawing.Color]::FromArgb(255, 92, 142, 255)), $thickness
$bluePen.StartCap = [System.Drawing.Drawing2D.LineCap]::Round
$bluePen.EndCap = [System.Drawing.Drawing2D.LineCap]::Round
$g.DrawLine($bluePen, $bottomX, $bottomY, $rightTopX, $rightTopY)

# === 3. 전구 (둥근 머리 + 짧은 base) ===
$bulbCx = 128 * $s
$bulbCy = 70 * $s
$bulbR = 30 * $s

# 전구 배경 (옅은 파랑)
$bulbBrush = New-Object System.Drawing.Drawing2D.LinearGradientBrush(
    (New-Object System.Drawing.PointF ($bulbCx - $bulbR), ($bulbCy - $bulbR)),
    (New-Object System.Drawing.PointF ($bulbCx + $bulbR), ($bulbCy + $bulbR)),
    [System.Drawing.Color]::FromArgb(255, 175, 200, 255),
    [System.Drawing.Color]::FromArgb(255, 135, 170, 240)
)
$g.FillEllipse($bulbBrush, ($bulbCx - $bulbR), ($bulbCy - $bulbR), ($bulbR * 2), ($bulbR * 2))

# 전구 base (소켓)
$socketW = 22 * $s
$socketH = 14 * $s
$socketBrush = New-Object System.Drawing.SolidBrush ([System.Drawing.Color]::FromArgb(255, 60, 50, 110))
$g.FillRectangle($socketBrush, ($bulbCx - $socketW / 2), ($bulbCy + $bulbR - 2), $socketW, $socketH)

# 전구 base 라인 (필라멘트)
$socketLinePen = New-Object System.Drawing.Pen ([System.Drawing.Color]::FromArgb(255, 100, 90, 160)), 1.5
$linesY = $bulbCy + $bulbR + 2
$g.DrawLine($socketLinePen, ($bulbCx - $socketW / 2 + 3), $linesY, ($bulbCx + $socketW / 2 - 3), $linesY)
$g.DrawLine($socketLinePen, ($bulbCx - $socketW / 2 + 3), ($linesY + 4), ($bulbCx + $socketW / 2 - 3), ($linesY + 4))

# === 4. 전구 안 중괄호 {} ===
$braceFont = New-Object System.Drawing.Font ("Consolas", ($Size * 0.13), [System.Drawing.FontStyle]::Bold)
$braceBrush = New-Object System.Drawing.SolidBrush ([System.Drawing.Color]::FromArgb(255, 40, 30, 90))
$braceFmt = New-Object System.Drawing.StringFormat
$braceFmt.Alignment = [System.Drawing.StringAlignment]::Center
$braceFmt.LineAlignment = [System.Drawing.StringAlignment]::Center
$braceRect = New-Object System.Drawing.RectangleF ($bulbCx - $bulbR), ($bulbCy - $bulbR), ($bulbR * 2), ($bulbR * 2)
$g.DrawString("{ }", $braceFont, $braceBrush, $braceRect, $braceFmt)

# === 5. 별 2개 (전구 오른쪽 위) ===
function Draw-Star {
    param($graphics, $cx, $cy, $size, $color)
    $pts = New-Object 'System.Collections.Generic.List[System.Drawing.PointF]'
    for ($i = 0; $i -lt 10; $i++) {
        $angle = ($i * 36 - 90) * [Math]::PI / 180
        $r = if ($i % 2 -eq 0) { $size } else { $size * 0.4 }
        $pt = New-Object System.Drawing.PointF (($cx + [Math]::Cos($angle) * $r), ($cy + [Math]::Sin($angle) * $r))
        $pts.Add($pt)
    }
    $brush = New-Object System.Drawing.SolidBrush $color
    $graphics.FillPolygon($brush, $pts.ToArray())
}

# 큰 별
Draw-Star -graphics $g -cx (175 * $s) -cy (40 * $s) -size (10 * $s) -color ([System.Drawing.Color]::FromArgb(255, 230, 230, 255))
# 작은 별
Draw-Star -graphics $g -cx (192 * $s) -cy (62 * $s) -size (6 * $s) -color ([System.Drawing.Color]::FromArgb(255, 230, 230, 255))

# === 저장 ===
$g.Dispose()
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $out) | Out-Null
$bm.Save($out, [System.Drawing.Imaging.ImageFormat]::Png)
$bm.Dispose()

Write-Host "Created: $out ($Size x $Size)"
