\NeedsTeXFormat{LaTeX2e}
\ProvidesClass{utmreport}[2026/10/09 ELSE report profile and UTM thesis class]
% Canonical copy: shared/utmreport.cls.
% Every template folder carries its own copy so it compiles after being copied anywhere.
% After editing this file, run `python tools/check.py --sync` to refresh those copies.

\newif\ifUTMThesis
\DeclareOption{thesis}{\UTMThesistrue}
\ProcessOptions\relax

\LoadClass[12pt,a4paper]{report}

\ifUTMThesis
  % Current FCIM formatting rules, checked 9 October 2026.
  \RequirePackage[a4paper,top=2cm,bottom=2cm,left=2cm,right=1cm,headheight=0pt,headsep=0pt,footskip=7.3mm]{geometry}
\else
  % Measurements from the supplied ELSE mandatory report, not the conference template.
  \RequirePackage[a4paper,top=2cm,bottom=2cm,left=2cm,right=1cm,headheight=0pt,headsep=0pt,footskip=7.3mm]{geometry}
\fi
\RequirePackage{iftex}
\ifPDFTeX
  \RequirePackage[T1]{fontenc}
  \RequirePackage[utf8]{inputenc}
  \RequirePackage{mathptmx}
\else
  \RequirePackage{fontspec}
  \IfFontExistsTF{Times New Roman}{\setmainfont{Times New Roman}}{\setmainfont{Liberation Serif}}
\fi
\RequirePackage[romanian,english]{babel}
\RequirePackage{graphicx}
\RequirePackage{amsmath}
\RequirePackage{array}
\RequirePackage{tabularx}
\RequirePackage{longtable}
\RequirePackage{caption}
\RequirePackage{enumitem}
\RequirePackage{indentfirst}
\RequirePackage{setspace}
\RequirePackage{titlesec}
\RequirePackage{tocloft}
\RequirePackage{ragged2e}
\RequirePackage{listings}
\RequirePackage{xcolor}
\RequirePackage{etoolbox}
\RequirePackage{hyperref}
\hypersetup{hidelinks,hypertexnames=false,linktoc=page}
\makeatletter
\pdfstringdefDisableCommands{\let\MakeUppercase\@firstofone}
\makeatother

% Images can be referenced by file name alone when they are stored in figures/.
\graphicspath{{figures/}}

% Text still to be replaced is printed in red, as in the Word model.
% `python tools/check.py <your-report-folder>` lists any \placeholder left behind.
% The uppercase colour name keeps it working inside uppercased titles.
\definecolor{utmplaceholder}{rgb}{1,0,0}\colorlet{UTMPLACEHOLDER}{utmplaceholder}
\DeclareRobustCommand{\placeholder}[1]{\textcolor{utmplaceholder}{#1}}

% babel switches to its main language at \begin{document}; keep the language chosen
% in the preamble with \selectlanguage{english} or \selectlanguage{romanian} instead.
\makeatletter
\let\UTMBabelSelectLanguage\selectlanguage
\renewcommand{\selectlanguage}[1]{\main@language{#1}\UTMBabelSelectLanguage{#1}}
\AtBeginDocument{\let\selectlanguage\UTMBabelSelectLanguage}
\makeatother
\selectlanguage{romanian}
\setstretch{1.5}
\setlength{\parindent}{1.25cm}
\setlength{\parskip}{0pt}
\pagestyle{plain}

% Caption and heading names follow the selected language.
\addto\captionsenglish{%
  \renewcommand{\contentsname}{\ifUTMThesis Contents\else Table of contents\fi}%
  \renewcommand{\refname}{References}\renewcommand{\bibname}{References}%
  \renewcommand{\figurename}{Figure}\renewcommand{\tablename}{Table}}
\addto\captionsromanian{%
  \renewcommand{\contentsname}{Cuprins}%
  \renewcommand{\refname}{Bibliografie}\renewcommand{\bibname}{Bibliografie}%
  \renewcommand{\figurename}{Figura}\renewcommand{\tablename}{Tabelul}}

\ifUTMThesis
\titleformat{\chapter}[block]
  {\centering\bfseries\fontsize{13pt}{16pt}\selectfont}
  {\thechapter}
  {0.5em}
  {\MakeUppercase}
\titlespacing*{\chapter}{0pt}{0pt}{12pt}

\titleformat{\section}
  {\bfseries\fontsize{12pt}{14pt}\selectfont}
  {\thesection}
  {1em}
  {}
\titlespacing*{\section}{1.25cm}{12pt}{0pt}

\titleformat{\subsection}
  {\bfseries\fontsize{12pt}{14pt}\selectfont}
  {\thesubsection}
  {1em}
  {}
\titlespacing*{\subsection}{1.25cm}{12pt}{0pt}
\else
  % "1 CHAPTER TITLE": new page, centered, uppercase, bold 13 pt, one space after the number.
  \titleformat{\chapter}[block]
    {\centering\bfseries\fontsize{13pt}{16pt}\selectfont}
    {\thechapter}{0.5em}{\MakeUppercase}
  \titlespacing*{\chapter}{0pt}{-26pt}{18pt}
  % "1.1 Subchapter title": bold 12 pt, indented 1.25 cm.
  \titleformat{\section}{\bfseries\fontsize{12pt}{14pt}\selectfont}{\thesection}{0.5em}{}
  \titlespacing*{\section}{1.25cm}{12pt}{6pt}
  \titleformat{\subsection}{\bfseries\fontsize{12pt}{14pt}\selectfont}{\thesubsection}{0.5em}{}
  \titlespacing*{\subsection}{1.25cm}{12pt}{6pt}
\fi

\setcounter{secnumdepth}{2}
\setcounter{tocdepth}{2}
\renewcommand{\cftchapleader}{\cftdotfill{\cftdotsep}}
\renewcommand{\cftsecleader}{\cftdotfill{\cftdotsep}}
\ifUTMThesis
  \renewcommand{\cfttoctitlefont}{\hfill\bfseries\fontsize{14pt}{16pt}\selectfont}
  \renewcommand{\cftaftertoctitle}{\hfill}
\else
  % The Word model's contents: plain (not bold) entries, dense dot leaders,
  % chapters in uppercase, subchapters and appendices indented 0.5 cm.
  \renewcommand{\cftchapfont}{\normalfont}
  \renewcommand{\cftchappagefont}{\normalfont}
  \renewcommand{\cftdotsep}{0.6}
  \setlength{\cftchapnumwidth}{1.1em}
  \setlength{\cftsecindent}{0.5cm}
  \setlength{\cftsecnumwidth}{1.9em}
  \setlength{\cftsubsecindent}{1cm}
  \setlength{\cftsubsecnumwidth}{2.6em}
  % The contents title uses the same heading style as the other unnumbered chapters.
  \makeatletter
  \AtBeginDocument{\renewcommand{\tableofcontents}{\UTMUnnumberedChapter*{\contentsname}\@starttoc{toc}}}
  \makeatother
\fi
\setlength{\cftbeforechapskip}{0pt}

\setlist[itemize]{label=-,leftmargin=1.9cm,labelindent=1.25cm,labelsep=*,align=left,itemsep=0pt,topsep=0pt,parsep=0pt,partopsep=0pt}
\setlist[enumerate]{leftmargin=1.9cm,itemsep=0pt,topsep=0pt,parsep=0pt,partopsep=0pt}
\newenvironment{utmenumerate}{\begin{enumerate}[label=\alph*)]}{\end{enumerate}}

\numberwithin{equation}{chapter}
\numberwithin{figure}{chapter}
\numberwithin{table}{chapter}
% Current FCIM rules and the supplied ELSE model use bold captions, chapter-level
% numbering, centered figures, right-aligned table titles and 11 pt single-spaced tables.
\DeclareCaptionLabelSeparator{utmDash}{ \textendash\ }
\captionsetup[figure]{position=bottom,labelsep=utmDash,justification=centering,singlelinecheck=false,font={bf,normalsize},skip=4pt}
\captionsetup[table]{position=top,labelsep=utmDash,justification=raggedleft,singlelinecheck=false,font={bf,normalsize},skip=4pt}
\AtBeginEnvironment{tabular}{\fontsize{11pt}{13pt}\selectfont\setstretch{1}}
\AtBeginEnvironment{tabularx}{\fontsize{11pt}{13pt}\selectfont\setstretch{1}}
\AtBeginEnvironment{longtable}{\fontsize{11pt}{13pt}\selectfont\setstretch{1}}

\newcolumntype{Y}{>{\RaggedRight\arraybackslash}X}
\renewcommand{\arraystretch}{1.2}

\lstdefinestyle{utmcode}{
  basicstyle=\ttfamily\fontsize{10pt}{12pt}\selectfont,
  breaklines=true,
  frame=single,
  numbers=left,
  numberstyle=\tiny,
  tabsize=2,
  backgroundcolor=\color{gray!8}
}
\lstset{style=utmcode}

\ifPDFTeX
  \RequirePackage{courier}
  \newcommand{\UTMCodeFont}{\fontfamily{pcr}\selectfont}
\else
  \IfFontExistsTF{Courier New}{\newfontfamily\UTMCodeFont{Courier New}}{\newcommand{\UTMCodeFont}{\ttfamily}}
\fi
% Source code uses Courier New 10 pt with single line spacing in reports and theses.
\lstdefinestyle{elsereportcode}{
  basicstyle=\UTMCodeFont\fontsize{10pt}{11.5pt}\selectfont,
  breaklines=true,frame=none,numbers=none,backgroundcolor=\color{white},
  keepspaces=true,columns=fullflexible,showstringspaces=false
}
\lstset{style=elsereportcode}
\AtBeginEnvironment{lstlisting}{\setstretch{1}}

% Preliminary pages count, but their page numbers are not printed.
\newif\ifUTMFrontMatter
\UTMFrontMattertrue
\makeatletter
\def\ps@plain{\let\@mkboth\@gobbletwo
  \let\@oddhead\@empty\let\@evenhead\@empty
  \def\@oddfoot{\ifUTMFrontMatter\else\hfil\thepage\hfil\fi}
  \let\@evenfoot\@oddfoot}
\makeatother
\pagestyle{plain}

% Unnumbered chapter with a contents entry (uppercase in the report contents).
% The starred form omits the contents entry.
\makeatletter
\newcommand{\UTMUnnumberedChapter}{\@ifstar{\UTM@unnumbered\@gobble}{\UTM@unnumbered\@firstofone}}
\newcommand{\UTM@unnumbered}[2]{%
  \chapter*{#2}\phantomsection
  #1{\ifUTMThesis\addcontentsline{toc}{chapter}{#2}%
     \else\addcontentsline{toc}{chapter}{\protect\MakeUppercase{#2}}\fi}}
\makeatother
% Public name, e.g. \unnumberedchapter{Internship Summary Report}.
\newcommand{\unnumberedchapter}{\UTMUnnumberedChapter}
\newcommand{\utmfrontmatter}{\UTMFrontMattertrue}
\newcommand{\utmmainmatter}{\clearpage\UTMFrontMatterfalse\pagestyle{plain}}
\newcommand{\utmintroduction}{\utmmainmatter\UTMUnnumberedChapter{\iflanguage{english}{Introduction}{Introducere}}}
\newcommand{\utmconclusions}{\UTMUnnumberedChapter{\iflanguage{english}{Conclusions}{Concluzii}}}

\makeatletter
% Defined outside the profile switch so that TeX can skip the nested conditionals below.
\newif\ifUTMAppendixMode
\newif\ifUTMFirstAppendix
\ifUTMThesis
  % Thesis references follow numeric IEEE-style citations in square brackets.
  \renewenvironment{thebibliography}[1]{%
    \UTMUnnumberedChapter{\bibname}%
    \begin{list}{[\arabic{enumiv}]}{%
      \usecounter{enumiv}\setlength{\leftmargin}{1.5em}%
      \setlength{\labelwidth}{1.5em}\setlength{\labelsep}{0.5em}%
      \setlength{\itemindent}{0pt}\setlength{\itemsep}{0pt}%
      \setlength{\parsep}{0pt}\setlength{\topsep}{0pt}%
      \renewcommand{\makelabel}[1]{##1\hfil}}%
    \sloppy\clubpenalty4000\widowpenalty4000\sfcode`\.\@m}
    {\end{list}}
  % Appendices use a separate centered label and title, as in the current rules.
  \renewcommand{\appendix}{%
    \UTMUnnumberedChapter{\iflanguage{english}{Appendices}{Anexe}}%
    \setcounter{chapter}{0}\renewcommand{\thechapter}{\Alph{chapter}}%
    \titleformat{\chapter}[display]
      {\centering\bfseries\fontsize{13pt}{16pt}\selectfont}
      {\iflanguage{english}{APPENDIX}{ANEXA}~\thechapter}
      {0pt}{\MakeUppercase}%
    \titlespacing*{\chapter}{0pt}{0pt}{12pt}}
\else
  % Numbered chapters are listed in uppercase in the contents, as in the Word model.
  \let\UTMOriginalChapter\chapter
  \RenewDocumentCommand{\chapter}{s o m}{%
    \IfBooleanTF{#1}{\UTMOriginalChapter*{#3}}{%
      \ifUTMAppendixMode\UTMAppendixHeading{\IfValueTF{#2}{#2}{#3}}{#3}%
      \else\UTMOriginalChapter[\protect\MakeUppercase{\IfValueTF{#2}{#2}{#3}}]{#3}\fi}}

  % References: "REFERENCES" heading listed in the contents, entries numbered "1.".
  \renewenvironment{thebibliography}[1]{%
    \UTMUnnumberedChapter{\refname}%
    \begin{list}{\arabic{enumiv}.}{%
      \usecounter{enumiv}\setlength{\leftmargin}{0pt}\setlength{\labelwidth}{0.75em}%
      \setlength{\labelsep}{0.25em}\setlength{\itemindent}{1em}%
      \setlength{\itemsep}{0pt}\setlength{\parsep}{0pt}\setlength{\topsep}{0pt}%
      \renewcommand{\makelabel}[1]{##1\hfil}}%
    \sloppy\clubpenalty4000\widowpenalty4000\sfcode`\.\@m}
    {\end{list}}

  % Appendices: "APPENDICES" once, then "APPENDIX A – Title" centered, each on a new page.
  \newcommand{\UTMAppendixWord}{\iflanguage{english}{APPENDIX}{ANEXA}}
  \renewcommand{\appendix}{%
    \UTMUnnumberedChapter{\iflanguage{english}{Appendices}{Anexe}}%
    \setcounter{chapter}{0}\renewcommand{\thechapter}{\Alph{chapter}}%
    \UTMAppendixModetrue\UTMFirstAppendixtrue}
  \newcommand{\UTMAppendixHeading}[2]{%
    \ifUTMFirstAppendix\global\UTMFirstAppendixfalse\else\clearpage\fi
    \refstepcounter{chapter}\phantomsection
    \addcontentsline{toc}{section}{\UTMAppendixWord~\thechapter\ \textendash\ #1}%
    {\centering\bfseries\fontsize{12pt}{14pt}\selectfont
     \UTMAppendixWord~\thechapter\ \textendash\ #2\par}%
    \vspace{12pt}\@afterheading}
\fi
\makeatother

% Optional list of abbreviations, placed before the Introduction.
\newenvironment{utmabbreviations}
  {\UTMUnnumberedChapter{\iflanguage{english}{List of abbreviations}{Lista abrevierilor}}%
   \begin{description}[style=sameline,leftmargin=3cm,labelwidth=2.6cm,font=\normalfont\bfseries,itemsep=0pt,topsep=0pt]}
  {\end{description}}

\newcommand{\UTMLogoPath}{utm-logo.jpeg}
\newcommand{\utmlogo}[1]{\renewcommand{\UTMLogoPath}{#1}}

\newcommand{\institution}[1]{\gdef\UTMInstitution{#1}}
\newcommand{\faculty}[1]{\gdef\UTMFaculty{#1}}
\newcommand{\department}[1]{\gdef\UTMDepartment{#1}}
\newcommand{\reporttitle}[1]{\gdef\UTMReportTitle{#1}}
\newcommand{\reportsubtitle}[1]{\gdef\UTMReportSubtitle{#1}}
\newcommand{\student}[2]{\gdef\UTMStudentName{#1}\gdef\UTMStudentGroup{#2}}
\newcommand{\supervisor}[2]{\gdef\UTMSupervisorName{#1}\gdef\UTMSupervisorTitle{#2}}
\newcommand{\consultant}[2]{\gdef\UTMConsultantName{#1}\gdef\UTMConsultantTitle{#2}}
\newcommand{\cityyear}[2]{\gdef\UTMCity{#1}\gdef\UTMYear{#2}}
\newcommand{\documenttype}[1]{\gdef\UTMDocumentType{#1}}
\newcommand{\studyprogramme}[1]{\gdef\UTMStudyProgramme{#1}}
\newcommand{\team}[1]{\gdef\UTMTeam{#1}}
\newcommand{\submissiondate}[1]{\gdef\UTMSubmissionDate{#1}}
\newcommand{\internshipmentor}[1]{\gdef\UTMInternshipMentor{#1}}
\makeatletter
\newcommand{\teammember}[3]{\g@addto@macro\UTMTeamRows{\textbf{#1:} & #2, #3 & \UTMSignatureCell\\\hline}}
\makeatother
\newcommand{\UTMTeamRows}{}
\newcommand{\UTMSignatureCell}{\textit{*\iflanguage{english}{Signature}{Semnătura}}}
\newcommand{\UTMStudentLabel}{\iflanguage{english}{Student}{Student}}
\newcommand{\UTMGroupLabel}{\iflanguage{english}{Group}{Grupa}}
\newcommand{\UTMCoordinatorLabel}{\iflanguage{english}{Coordinator}{Coordonator}}
\newcommand{\UTMSupervisorDisplay}{\ifx\UTMSupervisorTitle\empty\else\UTMSupervisorTitle, \fi\UTMSupervisorName}

\institution{Universitatea Tehnică a Moldovei}
\faculty{Facultatea}
\department{Departamentul}
\reporttitle{Titlul lucrării}
\reportsubtitle{}
\student{Prenumele NUMELE}{grupa}
\supervisor{Prenumele NUMELE}{titlul științific}
\consultant{}{}
\cityyear{Chișinău}{2026}
\documenttype{Raport}
\studyprogramme{}
\team{Team number or name}
\submissiondate{....... / ....... / ..............}
\internshipmentor{From internship contract}

% Centered logo above the university, faculty, department and programme.
\newcommand{\UTMReportInstitutionBlock}{%
  \vspace*{20pt}%
  \begingroup
    \centering\linespread{1}\selectfont
    \includegraphics[height=2.5cm]{\UTMLogoPath}\par
    \vspace{0.5cm}%
    {\bfseries\fontsize{14pt}{17pt}\selectfont\MakeUppercase{\UTMInstitution}\par}
    {\fontsize{13pt}{16pt}\selectfont\UTMFaculty\par\UTMDepartment\par
    \ifx\UTMStudyProgramme\empty\else\UTMStudyProgramme\par\fi}
  \endgroup
}
% Document type (20 pt) and title (18 pt), both bold and uppercase.
\newcommand{\UTMReportHeadline}{%
  \vspace{3.4cm}
  {\bfseries\fontsize{20pt}{24pt}\selectfont\MakeUppercase{\UTMDocumentType}\par}
  \vspace{0.1cm}
  {\bfseries\fontsize{18pt}{22pt}\selectfont\MakeUppercase{\UTMReportTitle}\par}
  \ifx\UTMReportSubtitle\empty\else
    \vspace{0.4cm}{\bfseries\fontsize{14pt}{17pt}\selectfont \UTMReportSubtitle\par}
  \fi
}

\newcommand{\makeutmcover}{%
  \begin{titlepage}
  \centering
  \includegraphics[height=2cm]{\UTMLogoPath}\par\vspace{0.5cm}
  {\bfseries\fontsize{16pt}{18pt}\selectfont \UTMInstitution\par}
  \vspace{4cm}
  {\bfseries\fontsize{22pt}{26pt}\selectfont \UTMReportTitle\par}
  \ifx\UTMReportSubtitle\empty\else
    \vspace{0.5cm}{\bfseries\fontsize{16pt}{18pt}\selectfont \UTMReportSubtitle\par}
  \fi
  \vspace{3cm}
  \hfill\begin{minipage}{0.65\textwidth}
  \raggedright\bfseries\fontsize{16pt}{18pt}\selectfont
  \iflanguage{english}{Student}{Student}: \UTMStudentName\\
  \iflanguage{english}{Group}{Grupa}: \UTMStudentGroup\\[0.5cm]
  \iflanguage{english}{Supervisor}{Conducător}: \UTMSupervisorTitle, \UTMSupervisorName
  \end{minipage}
  \vfill
  {\bfseries\fontsize{16pt}{18pt}\selectfont \UTMCity{} - \UTMYear\par}
  \end{titlepage}
  \setcounter{UTMThesisTitlePageCount}{0}
}

% Individual report: the internship cover rhythm, with a right-aligned author block.
\newcommand{\makeutmreporttitlepage}{%
  \begin{titlepage}
  \centering
  \UTMReportInstitutionBlock
  \UTMReportHeadline
  \vspace{3cm}
  \hfill\begin{minipage}{0.62\textwidth}
  \raggedleft\fontsize{12pt}{14pt}\selectfont\setstretch{1.5}
  \textbf{\UTMStudentLabel:} \UTMStudentName\\
  \textbf{\UTMGroupLabel:} \UTMStudentGroup\\
  \textbf{\UTMCoordinatorLabel:} \UTMSupervisorDisplay
  \end{minipage}
  \vfill
  {\bfseries\fontsize{12pt}{14pt}\selectfont \UTMCity, \UTMYear\par}
  \end{titlepage}
  \setcounter{page}{2}\utmfrontmatter
}

% Team internship report: the mandatory ELSE cover.
\newcommand{\makeutminternshiptitlepage}{%
  \begin{titlepage}\centering
  \UTMReportInstitutionBlock
  \UTMReportHeadline
  \vspace{1.9cm}
  {\fontsize{13pt}{16pt}\selectfont\UTMTeam\par}
  \vspace{0.6cm}
  {\setstretch{1}\raggedright\textbf{\iflanguage{english}{Team members}{Membrii echipei}:}\par}
  \vspace{0.4cm}
  {\setstretch{1}\renewcommand{\arraystretch}{1.15}
  \begin{tabularx}{\textwidth}{|p{0.31\textwidth}|>{\raggedleft\arraybackslash}X|p{0.31\textwidth}|}\hline
  \UTMTeamRows
  \textbf{University Internship Mentor} & \UTMInternshipMentor & \UTMSignatureCell\\\hline
  \end{tabularx}\par}
  \vspace{0.5cm}
  {\setstretch{1}\fontsize{10pt}{12pt}\selectfont\itshape\justifying\noindent
  *By signing the cover page of this report, each student confirms their active involvement in both the development of the project and the writing of the report. Students who did not contribute meaningfully are not permitted to sign.\par}
  \vfill
  {\setstretch{1}Submission date: \UTMSubmissionDate\par}
  \vspace{2.1cm}
  {\setstretch{1}\bfseries\UTMCity, \UTMYear\par}
  \vspace{0.6cm}
  \end{titlepage}\setcounter{page}{2}\utmfrontmatter
}

\newcounter{UTMThesisTitlePageCount}
\newcommand{\makeutmthesistitlepage}{%
  \begin{titlepage}
  \centering
  {\bfseries\fontsize{12pt}{14pt}\selectfont
  \iflanguage{english}{Ministry of Education and Research of the Republic of Moldova}{Ministerul Educației și Cercetării al Republicii Moldova}\\
  \UTMInstitution\\
  \UTMFaculty\\
  \UTMDepartment\par}
  \vspace{1.5cm}
  \hfill\begin{minipage}{0.45\textwidth}
  \raggedright\bfseries\fontsize{11pt}{13pt}\selectfont
  \iflanguage{english}{Admitted to defense}{Admis la susținere}\\
  \iflanguage{english}{Head of department}{Șef departament}: \rule{4.5cm}{0.4pt}\\
  ``\rule{0.8cm}{0.4pt}''\rule{2.5cm}{0.4pt} \UTMYear
  \end{minipage}
  \vspace{2cm}
  {\bfseries\fontsize{20pt}{24pt}\selectfont \UTMReportTitle\par}
  \vspace{0.7cm}
  {\bfseries\fontsize{16pt}{19pt}\selectfont \UTMDocumentType\par}
  \vspace{2cm}
  \hfill\begin{minipage}{0.72\textwidth}
  \raggedright\bfseries\fontsize{12pt}{14pt}\selectfont
  \iflanguage{english}{Student}{Student}: \rule{2.4cm}{0.4pt} \UTMStudentName, \UTMStudentGroup\\[4pt]
  \iflanguage{english}{Supervisor}{Conducător}: \rule{2.4cm}{0.4pt} \UTMSupervisorTitle, \UTMSupervisorName\\[4pt]
  \ifx\UTMConsultantName\empty\else \iflanguage{english}{Consultant}{Consultant}: \rule{2.4cm}{0.4pt} \UTMConsultantTitle, \UTMConsultantName\\[4pt]\fi
  \end{minipage}
  \vfill
  {\bfseries\fontsize{12pt}{14pt}\selectfont \UTMCity{} - \UTMYear\par}
  \end{titlepage}
  % The titlepage environment resets its counter; retain all title pages in pagination.
  \stepcounter{UTMThesisTitlePageCount}
  \setcounter{page}{\numexpr2+\value{UTMThesisTitlePageCount}\relax}\utmfrontmatter
}

\newenvironment{utmabstract}[1][Rezumat]
  {\UTMUnnumberedChapter{#1}}
  {\clearpage}

% Keywords start their own indented paragraph: "Keywords: five to seven terms".
\newcommand{\utmkeywords}[1]{\par\textbf{\iflanguage{english}{Keywords}{Cuvinte cheie}:} #1\par}
