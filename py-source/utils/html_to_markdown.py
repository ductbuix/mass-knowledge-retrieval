from html import unescape
from html.parser import HTMLParser
from typing import List, Optional


class _HTMLToMarkdown(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.out: List[str] = []
        self.list_stack: List[str] = []  # 'ul' or 'ol'
        self.ol_counters: List[int] = []
        self.link_href: List[Optional[str]] = []
        self.in_pre = False
        self.in_code = False

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if tag in {"h1", "h2", "h3", "h4", "h5", "h6"}:
            level = int(tag[1])
            self.out.append("\n" + ("#" * level) + " ")
        elif tag == "p":
            self.out.append("\n\n")
        elif tag in {"strong", "b"}:
            self.out.append("**")
        elif tag in {"em", "i"}:
            self.out.append("*")
        elif tag == "a":
            self.out.append("[")
            self.link_href.append(attrs.get("href"))
        elif tag == "br":
            self.out.append("\n")
        elif tag == "ul":
            self.list_stack.append("ul")
        elif tag == "ol":
            self.list_stack.append("ol")
            self.ol_counters.append(0)
        elif tag == "li":
            indent = "  " * max(0, len(self.list_stack) - 1)
            if self.list_stack and self.list_stack[-1] == "ol":
                if self.ol_counters:
                    self.ol_counters[-1] += 1
                    num = self.ol_counters[-1]
                else:
                    num = 1
                self.out.append("\n" + indent + f"{num}. ")
            else:
                self.out.append("\n" + indent + "- ")
        elif tag == "pre":
            self.in_pre = True
            self.out.append("\n\n```\n")
        elif tag == "code":
            if self.in_pre:
                self.in_code = True
            else:
                self.out.append("`")
        elif tag == "img":
            alt = attrs.get("alt", "")
            src = attrs.get("src", "")
            self.out.append(f"![{alt}]({src})")

    def handle_endtag(self, tag):
        if tag in {"strong", "b"}:
            self.out.append("**")
        elif tag in {"em", "i"}:
            self.out.append("*")
        elif tag == "a":
            href = self.link_href.pop() if self.link_href else None
            if href:
                self.out.append(f"]({href})")
            else:
                self.out.append("]")
        elif tag in {"ul"}:
            if self.list_stack and self.list_stack[-1] == "ul":
                self.list_stack.pop()
            self.out.append("\n")
        elif tag in {"ol"}:
            if self.list_stack and self.list_stack[-1] == "ol":
                self.list_stack.pop()
            if self.ol_counters:
                self.ol_counters.pop()
            self.out.append("\n")
        elif tag == "pre":
            self.in_pre = False
            self.out.append("\n```\n\n")
        elif tag == "code":
            if self.in_pre:
                self.in_code = False
            else:
                self.out.append("`")

    def handle_data(self, data):
        # For simplicity, just append; newlines handled by tags
        if data:
            self.out.append(unescape(data))

    def get_markdown(self) -> str:
        text = "".join(self.out)
        # Normalize excessive blank lines
        text = text.replace("\r\n", "\n").replace("\r", "\n")
        return text.strip() + "\n"


def html_to_markdown(html: str) -> str:
    parser = _HTMLToMarkdown()
    try:
        parser.feed(html)
        parser.close()
    except Exception:
        # Fallback: return plain text without tags
        import re as _re

        text = _re.sub(r"<[^>]+>", "", html)
        return text
    return parser.get_markdown()
