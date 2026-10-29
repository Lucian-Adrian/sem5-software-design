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


body_tex = cite_first(body_tex, "Vlissides, 1994)", "gof")
body_tex = cite_first(body_tex, "even when several threads ask at the same moment", "lazy")
body_tex = cite_first(body_tex, "dependency injection container instead:", "di")
body_tex = cite_first(body_tex, "does the same for arrays", "arraypool")

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
\documenttype{{Laboratory work No. 1}}
\reporttitle{{Creational design patterns in Domi (C\#)}}
\student{{Lucian-Adrian Gavril}}{{FAF-241}}
\supervisor{{Daniel Istrati}}{{asist. univ.}}
\cityyear{{Chișinău}}{{2026}}
\begin{{document}}
\makeutmreporttitlepage
\tableofcontents
\begin{{utmabstract}}[Abstract]
This laboratory work applies the six creational design patterns to Domi, a service where people form groups to rent a home together. Builder assembles and validates a group step by step. Abstract Factory produces a group and a suggested renter that always match. Prototype copies a group's settings into a new group without its identity or members. Singleton holds the shared catalog defaults, Factory Method lets subclasses choose the group inside a fixed workflow, and Object Pool reuses buffers for group summaries. For each pattern the report explains the problem it solves, where it fits in Domi, the code, why a simpler approach is not enough, and its disadvantages, and connects the patterns to the SOLID principles from Laboratory work No.~0.

\utmkeywords{{creational patterns, builder, abstract factory, prototype, singleton, factory method, object pool}}
\end{{utmabstract}}

\begin{{utmabbreviations}}
\item[DI] dependency injection
\item[GoF] Gang of Four (Gamma, Helm, Johnson, Vlissides)
\item[ID] identifier
\item[JSON] JavaScript Object Notation
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
\bibitem{{lazy}} Microsoft. \textit{{Lazy<T> Class}}. Available online: \url{{https://learn.microsoft.com/dotnet/api/system.lazy-1}} (accessed on 3 November 2026).
\bibitem{{di}} Microsoft. \textit{{Dependency injection in .NET: service lifetimes}}. Available online: \url{{https://learn.microsoft.com/dotnet/core/extensions/dependency-injection}} (accessed on 3 November 2026).
\bibitem{{arraypool}} Microsoft. \textit{{ArrayPool<T> Class}}. Available online: \url{{https://learn.microsoft.com/dotnet/api/system.buffers.arraypool-1}} (accessed on 3 November 2026).
\end{{thebibliography}}
\end{{document}}
"""
(HERE / "report.tex").write_text(tex, encoding="utf-8")
print("report.tex written")
