"""Builds report.tex (UTM template) from report.md, the single content source."""
import re
import subprocess
import sys
from pathlib import Path

HERE = Path(__file__).parent
PANDOC = sys.argv[1]

md = (HERE / "report.md").read_text(encoding="utf-8")

# Drop the Markdown title block and contents list; the template makes its own.
md = md[md.index("## 1. Task and purpose"):]
md = md.replace("\n---\n", "\n")

# Image + italic caption line -> one captioned figure (PNG for LaTeX).
md = re.sub(
    r"!\[[^\]]*\]\(assets/([\w-]+)\.svg\)\s*\n\s*\n\*Figure \d+\. (.+?)\*",
    lambda m: f"![{m.group(2)}](assets/{m.group(1)}.png)",
    md,
)

# Strip manual numbering from headings; the template numbers chapters and sections.
md = re.sub(r"^(#{2,3}) \d+(\.\d+)*\.? ", r"\1 ", md, flags=re.M)

# Split into introduction, body chapters and conclusions.
parts = re.split(r"^## ", md, flags=re.M)[1:]
intro, body, concl = parts[0], parts[1:-1], parts[-1]


def to_latex(text: str) -> str:
    out = subprocess.run(
        [PANDOC, "-f", "gfm+implicit_figures", "-t", "latex", "--listings", "--wrap=none",
         "--top-level-division=chapter"],
        input=text, capture_output=True, text=True, encoding="utf-8", check=True,
    ).stdout
    out = out.replace("\\begin{figure}", "\\begin{figure}[H]")
    out = re.sub(r"\\includegraphics(\[[^\]]*\])?\{",
                 r"\\includegraphics[width=0.92\\textwidth,height=0.42\\textheight,keepaspectratio]{", out)
    out = re.sub(r"\\label\{[^}]*\}", "", out)
    out = out.replace("\\begin{lstlisting}[language=json]", "\\begin{lstlisting}")
    # Wrapping columns of equal width instead of single-line `l` columns.
    out = re.sub(
        r"\\begin\{longtable\}\[\]\{@\{\}(l+)@\{\}\}",
        lambda m: "\\begin{longtable}[]{@{}" + ">{\\raggedright\\arraybackslash}p{%.3f\\linewidth}" % (0.96 / len(m.group(1))) * len(m.group(1)) + "@{}}",
        out,
    )
    out = re.sub(r"\\item\n\s+", r"\\item ", out)
    out = out.replace("├── ", "  ").replace("└── ", "  ")
    # UTM rule: every enumeration item ends with a semicolon.
    out = re.sub(r"^(\s*\\item(?!\[).*?)[.,:;]?\s*$", r"\1;", out, flags=re.M)
    return out


def body_only(section: str) -> str:
    return section.split("\n", 1)[1]


# Chapter (#) > section (##): lift the Markdown ### headings one level.
intro, concl = (re.sub(r"^### ", "## ", s, flags=re.M) for s in (intro, concl))
body = [re.sub(r"^### ", "## ", s, flags=re.M) for s in body]


intro_tex = to_latex(body_only(intro)).replace("\\caption{", "\\caption*{Figure 1 -- ")  # unnumbered chapter
body_tex = to_latex("\n".join("# " + s for s in body))
concl_tex = to_latex(body_only(concl))


def cite_first(tex: str, anchor: str, key: str) -> str:
    i = tex.find(anchor)
    if i < 0:
        sys.exit(f"citation anchor not found: {anchor}")
    j = i + len(anchor)
    return tex[:j] + f"~\\cite{{{key}}}" + tex[j:]


body_tex = cite_first(body_tex, "Behavioral design patterns assign responsibilities and communication between objects", "gof")

tex = rf"""\documentclass{{utmreport}}
\usepackage{{float,needspace,booktabs}}
\providecommand{{\tightlist}}{{\setlength{{\itemsep}}{{0pt}}\setlength{{\parskip}}{{0pt}}}}
\providecommand{{\passthrough}}[1]{{#1}}
\providecommand{{\pandocbounded}}[1]{{#1}}
\newcounter{{none}}
\selectlanguage{{english}}
\institution{{Technical University of Moldova}}
\faculty{{Faculty of Computers, Informatics, and Microelectronics}}
\department{{Department of Software Engineering}}
\studyprogramme{{Software Engineering Study Programme}}
\documenttype{{Laboratory work No. 3}}
\reporttitle{{Behavioral design patterns in Domi (C\#)}}
\student{{Lucian-Adrian Gavril}}{{FAF-241}}
\supervisor{{Daniel Istrati}}{{asist. univ.}}
\cityyear{{Chișinău}}{{2026}}
\begin{{document}}
\makeutmreporttitlepage
\tableofcontents
\begin{{utmabstract}}[Abstract]
This laboratory work extends Domi, a shared-rent roommate service built in Laboratory works No. 1 and No. 2, with three behavioral design patterns in C\#: Strategy, Command and Observer. Strategy lets the group matcher switch filtering rules at runtime without conditionals inside the context. Command turns group creation and membership changes into executable, queueable and undoable request objects whose business rules remain in the domain and guarded service pipeline. Observer broadcasts successful state changes and undo operations to independent subscribers that attach and detach at runtime. All three behavioral patterns interact on a single execution path that also crosses the Lab 1 Builder and the Lab 2 Adapter, Facade, Proxy and Decorators, verified by 25 xUnit tests.

\utmkeywords{{behavioral patterns, strategy, command, observer, undo, solid}}
\end{{utmabstract}}

\begin{{utmabbreviations}}
\item[FIFO] first in, first out
\item[GoF] Gang of Four (Gamma, Helm, Johnson, Vlissides)
\item[ID] identifier
\item[LIFO] last in, first out
\item[SOLID] single responsibility, open/closed, Liskov substitution, interface segregation, dependency inversion
\item[UML] Unified Modeling Language
\end{{utmabbreviations}}

\utmintroduction
{intro_tex}
{body_tex}
\utmconclusions
{concl_tex}

\begin{{thebibliography}}{{9}}
\bibitem{{gof}} E. Gamma, R. Helm, R. Johnson, J. Vlissides. \textit{{Design Patterns: Elements of Reusable Object-Oriented Software}}. Addison-Wesley, 1994.
\end{{thebibliography}}
\end{{document}}
"""
(HERE / "report.tex").write_text(tex, encoding="utf-8")
print("report.tex written")
