"""Make static Isomorph SVG exports portable, then render PNG previews.
Run after render-diagrams.ts. Resolves line dashes and UML markers as geometry
because the installed MuPDF SVG importer does not preserve these features.
"""
from pathlib import Path
import math
import xml.etree.ElementTree as ET
import pymupdf
NS='http://www.w3.org/2000/svg'
ET.register_namespace('',NS)
def tag(n): return '{'+NS+'}'+n
for source in Path(__file__).parent.glob('*.svg'):
    root=ET.parse(source).getroot()
    for parent in list(root.iter()):
        for element in list(parent):
            if element.tag != tag('line'): continue
            dash=element.get('stroke-dasharray')
            marker=element.get('marker-end')
            if not dash and not marker: continue
            x1,y1,x2,y2=(float(element.get(k,'0')) for k in ('x1','y1','x2','y2'))
            length=math.hypot(x2-x1,y2-y1)
            if not length: continue
            ux,uy=(x2-x1)/length,(y2-y1)/length
            stroke=element.get('stroke','#52675f')
            if dash:
                on,off=map(float,dash.split(','))
                group=ET.Element(tag('g'))
                start=0
                while start<length:
                    end=min(start+on,length)
                    attrs={k:v for k,v in element.attrib.items() if k not in ('stroke-dasharray','marker-end','x1','x2','y1','y2')}
                    attrs.update(x1=str(x1+ux*start),y1=str(y1+uy*start),x2=str(x1+ux*end),y2=str(y1+uy*end))
                    ET.SubElement(group,tag('line'),attrs)
                    start+=on+off
                index=list(parent).index(element)
                parent.remove(element);parent.insert(index,group)
            else: element.attrib.pop('marker-end',None)
            if marker:
                hollow='hollow-arrow' in marker
                depth,width=(13,4.5) if hollow else (10,3.5)
                points=[(x2,y2),(x2-ux*depth-uy*width,y2-uy*depth+ux*width),(x2-ux*depth+uy*width,y2-uy*depth-ux*width)]
                ET.SubElement(parent,tag('polygon'),{'points':' '.join(f'{x:.2f},{y:.2f}' for x,y in points),'fill':'#ffffff' if hollow else stroke,'stroke':stroke,'stroke-width':'1.5'})
    data=ET.tostring(root,encoding='utf-8',xml_declaration=True)
    source.write_bytes(data)
    doc=pymupdf.open(stream=data,filetype='svg')
    doc[0].get_pixmap(matrix=pymupdf.Matrix(1.2,1.2)).save(source.with_suffix('.png'))
    print('PASS portable SVG/PNG:',source.name)
