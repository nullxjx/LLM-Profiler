from reportlab.lib.pagesizes import A4
from reportlab.lib import colors
from reportlab.platypus import SimpleDocTemplate, Paragraph, Table, TableStyle, Image, Flowable, Spacer
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.colors import blue
from reportlab.pdfgen.canvas import Canvas

# 创建一个临时的Canvas对象来测量文本
_temp_canvas = Canvas(None)


class HyperlinkedURL(Flowable):
    def __init__(self, url, text, style):
        Flowable.__init__(self)
        self.url = url
        self.text = text
        self.style = style

    def wrap(self, availWidth, availHeight):
        style = self.style
        _temp_canvas.setFont(style.fontName, style.fontSize)
        self.width, self.height = _temp_canvas.stringWidth(self.text, style.fontName, style.fontSize), style.leading
        return self.width, self.height

    def draw(self):
        canvas = self.canv
        style = self.style
        canvas.setFont(style.fontName, style.fontSize)
        canvas.setFillColor(blue)
        canvas.drawString(0, 0, self.text)
        canvas.linkURL(self.url, (0, 0, self.width, self.height), relative=1)


def create_peak_pdf(save_dir, title, metrics, images, config):
    """
    创建一个pdf文件
    :param save_dir: pdf文件保存路径
    :param title: pdf标题
    :param metrics: 指标，示例数据如下
    {
        "500": {
            "8": {
                "cos_url": "xxx",
                "cos_path": "xxx",
                "throughput": 158
            }
        }
    }
    :param images: 图片信息，示例数据如下
    [
    {
        "title": "Performance metrics vs throughput",
        "path": "visual_results/metrics_vs_concurrency.png"
    }
    ]
    :return:
    """
    # 创建一个PDF文档
    doc = SimpleDocTemplate(save_dir, pagesize=A4)

    # 定义样式
    styles = getSampleStyleSheet()
    title_style = ParagraphStyle(
        "title",
        fontSize=18,
        leading=24,
        spaceAfter=20,
        alignment=1,  # 设置居中对齐
    )
    text_style = styles["BodyText"]

    # 添加标题
    title = Paragraph(title, title_style)

    # 创建表格数据
    table_data = [
        ["input tokens", "output tokens", "cos url", "throughput (tokens/s)"],
    ]
    end = 1
    span_rows = []

    for input_tokens, value in metrics.items():
        start = end
        for max_new_tokens, data in value.items():
            first_column = ""
            if end == start:
                first_column = input_tokens
            table_data.append([
                first_column,
                max_new_tokens,
                HyperlinkedURL(data["cos_url"], "click me", text_style),
                data["throughput"]
            ])
            end += 1
        span_rows.append(("SPAN", (0, start), (0, end - 1)))

    # 创建表格，并设置样式
    table = Table(table_data, colWidths=[80, 80, 80, 120])
    table_style = [
        ("BACKGROUND", (0, 0), (-1, 0), colors.grey),
        ("TEXTCOLOR", (0, 0), (-1, 0), colors.whitesmoke),
        ("ALIGN", (0, 0), (-1, -1), "CENTER"),
        ("VALIGN", (0, 0), (-1, 0), "MIDDLE"),  # 设置表格标题行的垂直对齐为居中
        ("FONTNAME", (0, 0), (-1, 0), "Helvetica-Bold"),
        ("FONTSIZE", (0, 0), (-1, 0), 10),
        ("BOTTOMPADDING", (0, 0), (-1, 0), 12),
        ("BACKGROUND", (0, 1), (-1, -1), colors.beige),
        ("GRID", (0, 0), (-1, -1), 1, colors.black),
        ("FONTNAME", (0, 1), (-1, -1), "Helvetica"),
        ("FONTSIZE", (0, 1), (-1, -1), 12),
        ("BOTTOMPADDING", (0, 1), (-1, -1), 12),
        ("VALIGN", (0, 1), (0, -1), "MIDDLE"),  # 设置第一列的垂直对齐为居中
    ]
    table_style += span_rows
    table.setStyle(TableStyle(table_style))

    image_elements = []
    idx = 2
    for image in images:
        image_elements.append(Paragraph("{}. {}".format(idx, image['title']), text_style))
        image_elements.append(Image(image['path'], width=400, height=200))
        idx += 1

    # 创建 config 表格数据
    config_table_data = [[key, value] for key, value in config.items()]

    # 创建 config 表格，并设置样式
    config_table = Table(config_table_data, colWidths=[200, 200])
    config_table_style = [
        ("GRID", (0, 0), (-1, -1), 1, colors.black),
        ("FONTNAME", (0, 0), (-1, -1), "Helvetica"),
        ("FONTSIZE", (0, 0), (-1, -1), 12),
        ("ALIGN", (0, 0), (-1, -1), "LEFT"),
        ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 12),  # 增加每行的底部填充以增加行高
    ]
    config_table.setStyle(TableStyle(config_table_style))
    config_table_title = Paragraph("0. Configurations", text_style)

    spacer = Spacer(1, 10)  # 创建一个 Spacer 对象，用于在元素之间添加空白

    # 将元素添加到文档中
    table_title = Paragraph("1. throughput of input tokens and output tokens", text_style)
    all_elements = [title, config_table_title, spacer, config_table, spacer, table_title, spacer, table]
    all_elements += image_elements
    doc.build(all_elements)


def create_inline_pdf(save_dir, title, metrics, config):
    # 创建一个PDF文档
    doc = SimpleDocTemplate(save_dir, pagesize=A4)

    # 定义样式
    styles = getSampleStyleSheet()
    title_style = ParagraphStyle(
        "title",
        fontSize=18,
        leading=24,
        spaceAfter=20,
        alignment=1,  # 设置居中对齐
    )
    text_style = styles["BodyText"]

    # 添加标题
    title = Paragraph(title, title_style)

    # 创建表格数据
    table_data = [
        ["time limit (s)", "input/output tokens", "outputTokens/s", "inputTokens/s", "requests/s"],
    ]
    end = 1
    span_rows = []

    timeout = 0
    for data in metrics:
        if data['timeoutSeconds'] != timeout:
            timeout = data['timeoutSeconds']
            start = end
        first_column = ""
        if end == start:
            first_column = str(timeout) + " s"
        table_data.append([
            first_column,
            str(data['inputTokens']) + " / " + str(data['outputTokens']),
            data['outputTokensPerSeconds'],
            data['inputTokensPerSeconds'],
            data['requestPerSeconds']
        ])
        end += 1
        span_rows.append(("SPAN", (0, start), (0, end - 1)))

    # 创建表格，并设置样式
    table = Table(table_data, colWidths=[80, 100, 100, 100, 80])
    table_style = [
        ("BACKGROUND", (0, 0), (-1, 0), colors.grey),
        ("TEXTCOLOR", (0, 0), (-1, 0), colors.whitesmoke),
        ("ALIGN", (0, 0), (-1, -1), "CENTER"),
        ("VALIGN", (0, 0), (-1, 0), "MIDDLE"),  # 设置表格标题行的垂直对齐为居中
        ("FONTNAME", (0, 0), (-1, 0), "Helvetica-Bold"),
        ("FONTSIZE", (0, 0), (-1, 0), 10),
        ("BOTTOMPADDING", (0, 0), (-1, 0), 12),
        ("BACKGROUND", (0, 1), (-1, -1), colors.beige),
        ("GRID", (0, 0), (-1, -1), 1, colors.black),
        ("FONTNAME", (0, 1), (-1, -1), "Helvetica"),
        ("FONTSIZE", (0, 1), (-1, -1), 12),
        ("BOTTOMPADDING", (0, 1), (-1, -1), 12),
        ("VALIGN", (0, 1), (0, -1), "MIDDLE"),  # 设置第一列的垂直对齐为居中
    ]
    table_style += span_rows
    table.setStyle(TableStyle(table_style))

    # 创建 config 表格数据
    config_table_data = [[key, value] for key, value in config.items()]

    # 创建 config 表格，并设置样式
    config_table = Table(config_table_data, colWidths=[200, 200])
    config_table_style = [
        ("GRID", (0, 0), (-1, -1), 1, colors.black),
        ("FONTNAME", (0, 0), (-1, -1), "Helvetica"),
        ("FONTSIZE", (0, 0), (-1, -1), 12),
        ("ALIGN", (0, 0), (-1, -1), "LEFT"),
        ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 12),  # 增加每行的底部填充以增加行高
    ]
    config_table.setStyle(TableStyle(config_table_style))
    config_table_title = Paragraph("0. Configurations", text_style)

    spacer = Spacer(1, 10)  # 创建一个 Spacer 对象，用于在元素之间添加空白

    # 将元素添加到文档中
    table_title = Paragraph("1. throughput of input tokens and output tokens under time constraint", text_style)
    all_elements = [title, config_table_title, spacer, config_table, spacer, table_title, spacer, table]
    doc.build(all_elements)
