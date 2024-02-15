/*
Copyright 2023 Lee R. Boynton

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package html

import (
	"fmt"
	"strings"
	
	"github.com/boynton/api/common"
	"github.com/boynton/api/model"
	"github.com/boynton/api/smithy"
	"github.com/boynton/data"
)

const IndentAmount = "    "

type Generator struct {
	common.BaseGenerator
	ns              string
	name            string
	detailGenerator string
}

func (gen *Generator) Generate(schema *model.Schema, config *data.Object) error {
	err := gen.Configure(schema, config)
	if err != nil {
		return err
	}
	gen.ns = string(schema.ServiceNamespace())
	gen.name = string(schema.ServiceName())
	gen.detailGenerator = config.GetString("detail-generator") //should be either "smithy" or "
	gen.Begin()
	gen.GenerateHeader()
	gen.GenerateSummary()
	gen.GenerateOperations()
	gen.GenerateExceptions()
	gen.GenerateTypes()
	gen.GenerateFooter()
	s := gen.End()
	fname := gen.FileName(gen.name, ".md")
	err = gen.Write(s, fname, "")
	return err
}

func (gen *Generator) getDetailGenerator() common.Generator {
	dec := common.Decorator{
		BaseType: func(s string) string {
			return fmt.Sprintf("<em><strong>%s</strong></em>", s)
		},
		UserType: func(s string) string {
			return fmt.Sprintf("<em><strong><a href=\"#%s\">%s</a></strong></em>", strings.ToLower(s), s)
		},
	}
	switch gen.detailGenerator {
	case "smithy":
		g := new(smithy.IdlGenerator)
		g.Decorator = &dec
		return g
	default:
		g := new(common.ApiGenerator)
		g.Decorator = &dec
		return g
	}
}


func (gen *Generator) GenerateHeader() {
	gen.Emitf("<!DOCTYPE html>\n<html lang=\"US\">\n")
	gen.Emitf("<head>\n  <meta charset=\"utf-8\" />\n")
	gen.Emitf("  <meta http-equiv=\"X-UA-Compatible\" content=\"IE=edge,chrome=1\" />\n")
	gen.Emitf("  <title>%s</title>\n",  gen.name)
	gen.Emitf("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\" />\n")
	gen.Emitf("  <style id=\"mkstylesheet\">%s</style>\n", htmlStyle)
	gen.Emitf("  <style id=\"mkprintstyles\">@media print{#wrapper #generated-toc-clone,#generated-toc{display:none!important}html,body,#wrapper{font-size:10pt!important}}\n</style>\n")
	gen.Emit("</head>\n\n")
	gen.Emit("<body class=\"normal firstload\">\n  <div id=\"wrapper\">\n")
}

func (gen *Generator) GenerateFooter() {
	gen.Emitf("  </div>\n</body>\n</html>\n")
}

func (gen *Generator) GenerateSummary() {
	gen.Emitf("<h1 id=%q>TestService</h1>\n", gen.name)
	gen.Emitf("<p>%s</p>\n", gen.Schema.Comment) //common.FormatComment("", "", gen.Schema.Comment, 80, true)
	gen.Emitf("<ul>\n")
	if gen.name != "" {
		gen.Emitf("  <li><strong>Service</strong>: &ldquo;%s&rdquo;</li>\n", gen.name)
	}
	if gen.Schema.Version != "" {
		gen.Emitf("  <li><strong>Version</strong>: &ldquo;%s&rdquo;</li>\n", gen.Schema.Version)
	}
	if gen.ns != "" {
		gen.Emitf("  <li><strong>Namespace</strong>: &ldquo;%s&rdquo;</li>\n", gen.ns)
	}
	if gen.Schema.Base != "" {
		gen.Emitf("  <li><strong>Namespace</strong>: &ldquo;%s&rdquo;</li>\n", gen.Schema.Base)
	}
	gen.Emitf("</ul>\n")
	gen.Emitf("<h3 id=\"operations\">Operations</h3>\n")
	gen.Emitf("<ul>\n")
	for _, op := range gen.Operations() {
		sum := summarySignature(op)
		s := StripNamespace(op.Id)
		gen.Emitf("  <li><a href=\"#%s\">%s</a></li>\n", strings.ToLower(s), sum)
	}
	gen.Emitf("</ul>\n")
	gen.Emitf("<h3 id=\"types\">Types</h3>\n")
	gen.Emitf("<ul>\n")
	for _, td := range gen.Types() {
		//check if a type has input or output trait, if so, omit it here.
		s := StripNamespace(td.Id)
		gen.Emitf("  <li><a href=\"#%s\">%s</a> → <em>%s</em></li>\n", strings.ToLower(s), s, td.Base)
	}
	gen.Emit("</ul>\n\n")
}

func StripNamespace(target model.AbsoluteIdentifier) string {
	t := string(target)
	n := strings.Index(t, "#")
	if n < 0 {
		return t
	}
	return t[n+1:]
}

func ExplodeInputs(in *model.OperationInput) string {
	var types []string
	if in != nil {
		for _, f := range in.Fields {
			//types = append(types, string(f.Name) + " " + StripNamespace(f.Type))
			types = append(types, string(f.Name))
		}
		return strings.Join(types, ", ")
	}
	return ""
}

func ExplodeOutputs(out *model.OperationOutput) string {
	var types []string
	for _, f := range out.Fields {
		//types = append(types, string(f.Name) + " " + StripNamespace(f.Type))
		types = append(types, string(f.Name))
	}
	return strings.Join(types, ", ")
}

func summarySignature(op *model.OperationDef) string {
	in := ExplodeInputs(op.Input)
	out := ExplodeOutputs(op.Output)
	s := StripNamespace(op.Id)
	return fmt.Sprintf("%s(%s) → (%s)", s, in, out)
}

func (gen *Generator) generateApiOperation(op *model.OperationDef) string {
	g := gen.getDetailGenerator()
	conf := data.NewObject()
	err := g.Configure(gen.Schema, conf)
	if err != nil {
		return "Whoops: " + err.Error()
	}
	g.Begin()
	g.GenerateOperation(op)
	s := g.End()
	return s
}

func (gen *Generator) GenerateOperation(op *model.OperationDef) error {
	opId := StripNamespace(op.Id)
	gen.Emitf("<h3 id=%q>%s</h3>\n", strings.ToLower(opId), opId)
	gen.Emitf("<pre class=\"mknohighlight\"><code>\n")
	gen.Emitf("%s\n\n", gen.generateApiOperation(op))
	gen.Emitf("</code></pre>\n")
	return nil
}

func (gen *Generator) GenerateOperations() {
	//this is a high level signature without types or exceptions
	gen.Emitf("<h2 id=\"operations\">Operations</h2>\n")
	if len(gen.Schema.Operations) > 0 {
		for _, op := range gen.Operations() {
			gen.GenerateOperation(op)
		}
		gen.Emit("\n")
	}
}

func (gen *Generator) GenerateException(out *model.OperationOutput) error {
	s := StripNamespace(out.Id)
	gen.Emitf("<h3 id=%q>%s</h3>\n", strings.ToLower(s), s)
	gen.Emitf("<pre class=\"mknohighlight\"><code>\n")
	gen.Emitf("%s\n\n//FIX ME: Exception\n")
	gen.Emitf("</code></pre>\n")
	return nil
}

func (gen *Generator) GenerateType(td *model.TypeDef) error {
	s := StripNamespace(td.Id)
	gen.Emitf("<h3 id=%q>%s</h3>\n", strings.ToLower(s), s)
	gen.Emitf("<pre class=\"mknohighlight\"><code>\n")
	gen.Emitf("%s\n\n", gen.generateApiType(td))
	gen.Emitf("</code></pre>\n")
	return nil
}

func (gen *Generator) generateApiType(op *model.TypeDef) string {
	g := gen.getDetailGenerator()
	conf := data.NewObject()
	err := g.Configure(gen.Schema, conf)
	if err != nil {
		return "Whoops: " + err.Error()
	}
	g.Begin()
	g.GenerateType(op)
	s := g.End()
	return s
}

func (gen *Generator) GenerateExceptions() {
/*	emitted := make(map[model.AbsoluteIdentifier]*model.OperationOutput, 0)
	for _, op := range gen.Operations() {		
		for _, out := range op.Exceptions {
			if _, ok := emitted[out.Id]; ok {
				//duplicates?
			} else {
				if len(emitted) == 0 {
					gen.Emitf("<h2 id=\"exceptions\">Exceptions</h2>\n")
				}
				gen.GenerateException(out)
				emitted[out.Id] = out
			}
		}
		if len(emitted) > 0 {
			gen.Emit("\n")
		}
	}
*/	
}

func (gen *Generator) collectExceptionTypes() {
}

func (gen *Generator) GenerateTypes() {
	tds := gen.Schema.Types
	//emitted := make(map[string]bool, 0)
	
	if len(tds) > 0 {
		gen.Emitf("<h2 id=\"types\">Types</h2>\n")
		for _, td := range gen.Types() {
			gen.GenerateType(td)
		}
		gen.Emit("\n")
	}
}

var htmlStyle string = `
#wrapper{color:#24292e;font-family:-apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol';margin:0;text-size-adjust:100%}#wrapper aside,#wrapper article,#wrapper details,#wrapper figcaption,#wrapper figure,#wrapper footer,#wrapper header,#wrapper hgroup,#wrapper main,#wrapper menu,#wrapper nav,#wrapper section,#wrapper summary{display:block}#wrapper audio,#wrapper canvas,#wrapper progress,#wrapper video{display:inline-block;vertical-align:baseline}#wrapper audio:not([controls]){display:none;height:0}#wrapper [hidden],#wrapper template{display:none}#wrapper a{background-color:transparent}#wrapper a:active,#wrapper a:hover{outline:0}#wrapper abbr[title]{border-bottom:1px dotted}#wrapper b,#wrapper strong{font-weight:bold}#wrapper dfn{font-style:italic}#wrapper mark{background:#ff0;color:#000}#wrapper small{font-size:80%}#wrapper sub,#wrapper sup{font-size:75%;line-height:0;position:relative;vertical-align:baseline}#wrapper sup{top:-.5em}#wrapper sub{bottom:-.25em}#wrapper img{border:0}#wrapper svg:not(:root){overflow:hidden}#wrapper figure{margin:1em 40px}#wrapper hr{box-sizing:content-box;height:0}#wrapper pre{overflow:auto}#wrapper code,#wrapper kbd,#wrapper pre,#wrapper samp{font-family:monospace, monospace;font-size:1em}#wrapper button,#wrapper input,#wrapper optgroup,#wrapper select,#wrapper textarea{color:inherit;font:inherit;margin:0}#wrapper button{overflow:visible}#wrapper button,#wrapper select{text-transform:none}#wrapper button,#wrapper input[type='button'],#wrapper input[type='reset'],#wrapper input[type='submit']{-webkit-appearance:button;cursor:pointer}#wrapper button[disabled]{cursor:default}#wrapper button::-moz-focus-inner,#wrapper input::-moz-focus-inner{border:0;padding:0}#wrapper input{line-height:normal}#wrapper input[type='checkbox'],#wrapper input[type='radio']{box-sizing:border-box;padding:0}#wrapper input[type='number']::-webkit-inner-spin-button,#wrapper input[type='number']::-webkit-outer-spin-button{height:auto}#wrapper input[type='search']{-webkit-appearance:textfield;box-sizing:content-box}#wrapper input[type='search']::-webkit-search-cancel-button,#wrapper input[type='search']::-webkit-search-decoration{-webkit-appearance:none}#wrapper fieldset{border:1px solid #c0c0c0;margin:0 2px;padding:0.35em 0.625em 0.75em}#wrapper legend{border:0;padding:0}#wrapper textarea{overflow:auto}#wrapper optgroup{font-weight:bold}#wrapper table{border-collapse:collapse;border-spacing:0}#wrapper *{box-sizing:border-box}#wrapper input,#wrapper select,#wrapper textarea,#wrapper button{font:14px/21px -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol'}#wrapper body{font:14px/21px -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol';color:#333;background-color:#fff}#wrapper a{color:#4078c0;text-decoration:none}#wrapper a:hover,#wrapper a:active{text-decoration:underline}#wrapper hr,#wrapper .rule{background:transparent;border:0;border-bottom:1px solid #ddd;height:0;margin:15px 0;overflow:hidden}#wrapper hr:before,#wrapper .rule:before{content:'';display:table}#wrapper hr:after,#wrapper .rule:after{clear:both;content:'';display:table}#wrapper h1,#wrapper h2,#wrapper h3,#wrapper h4,#wrapper h5,#wrapper h6{font-weight:600;line-height:1.1;margin:24px 0 16px;padding:0}#wrapper h1,#wrapper h2{border-bottom:1px solid #eaecef}#wrapper h1{font-size:32px;line-height:40px;margin:0 0 16px;padding:0 0 9.600000381469727px}#wrapper h2{font-size:24px;line-height:30px;padding:0 0 7.199999809265137px}#wrapper h3{font-size:20px;line-height:25px}#wrapper h4{font-size:16px;line-height:20px;margin:24px 0 16px;padding:0}#wrapper h5{font-size:14px;line-height:17px}#wrapper h6{font-size:13.600000381469727px;line-height:17px}#wrapper small{font-size:90%}#wrapper blockquote{margin:0}#wrapper ol ol,#wrapper ul ol{list-style-type:lower-roman}#wrapper ul ul ol,#wrapper ul ol ol,#wrapper ol ul ol,#wrapper ol ol ol{list-style-type:lower-alpha}#wrapper dd{margin-left:0}#wrapper tt,#wrapper code,#wrapper pre{font-family:SFMono-Regular,Consolas,Liberation Mono,Menlo,Courier,monospace}#wrapper pre{background-color:#f6f8fa;border-radius:3px;font-size:85%;line-height:1.45;overflow:auto;padding:16px;margin-top:0;margin-bottom:0}#wrapper{-webkit-font-smoothing:antialiased;background:#fff;border:solid 1px #dddddd !important;border-radius:3px;color:#333;font:14px -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol';line-height:1.6;padding:3px}p{margin:1em 0}a{color:#4183c4;text-decoration:none}#wrapper{background-color:#fff;font-size:16px;line-height:1.6;margin:15px;padding:30px}#wrapper>*:first-child{margin-top:0 !important}#wrapper>*:last-child{margin-bottom:0 !important}@media screen{#wrapper{border:solid 1px #ddd}}p,blockquote,ul,ol,dl,table,pre{margin-bottom:16px}hr{height:4px;padding:0;margin:16px 0;background-color:#e7e7e7;border:0 none}ul,ol{padding-left:2.8rem}ul.no-list,ol.no-list{padding:0;list-style-type:none}ul ul,ul ol{margin-top:0;margin-bottom:0}ol ol,ol ul{margin-top:0;margin-bottom:0}li>p{margin-bottom:0}li p+p{margin-top:16px}dl{padding:0}dl dt{padding:0;margin-top:16px;font-size:1em;font-style:italic;font-weight:700}dl dd{padding:0 16px;margin-bottom:16px}blockquote{padding:0 15px;margin-left:0;color:#777;border-left:4px solid #ddd}blockquote>:first-child{margin-top:0}blockquote>:last-child{margin-bottom:0}table{display:block;width:100%;overflow:auto}table th{font-weight:700;padding:6px 13px;border:1px solid #ddd}table td{padding:6px 13px;border:1px solid #ddd}table tr{background-color:#fff;border-top:1px solid #ccc}table tr:nth-child(2n){background-color:#f8f8f8}img{max-width:100%;-moz-box-sizing:border-box;box-sizing:border-box}span.frame{display:block;overflow:hidden}span.frame>span{display:block;float:left;width:auto;padding:7px;margin:13px 0 0;overflow:hidden;border:1px solid #ddd}span.frame span img{display:block;float:left}span.frame span span{display:block;padding:5px 0 0;clear:both;color:#333}span.align-center{display:block;overflow:hidden;clear:both}span.align-center>span{display:block;margin:13px auto 0;overflow:hidden;text-align:center}span.align-center span img{margin:0 auto;text-align:center}span.align-right{display:block;overflow:hidden;clear:both}span.align-right>span{display:block;margin:13px 0 0;overflow:hidden;text-align:right}span.align-right span img{margin:0;text-align:right}span.float-left{display:block;float:left;margin-right:13px;overflow:hidden}span.float-left span{margin:13px 0 0}span.float-right{display:block;float:right;margin-left:13px;overflow:hidden}span.float-right>span{display:block;margin:13px auto 0;overflow:hidden;text-align:right}code,tt{background-color:rgba(0,0,0,0.04);border-radius:3px;font-size:85%;margin:0;padding-bottom:.2em;padding-top:.2em;padding:0}code::before,code::after{content:'\00a0';letter-spacing:-.2em}tt:before,tt:after{content:'\00a0';letter-spacing:-.2em}code br,tt br{display:none}del code{text-decoration:inherit;vertical-align:text-top}pre>code{padding:0;margin:0;font-size:100%;white-space:pre;background:transparent;border:0}.highlight{margin-bottom:16px}.highlight pre{padding:16px;margin-bottom:0;overflow:auto;font-size:85%;line-height:1.45;background-color:#f7f7f7;border-radius:3px}pre{padding:16px;margin-bottom:16px;overflow:auto;font-size:85%;line-height:1.45;background-color:#f7f7f7;border-radius:3px;word-wrap:normal}pre code,pre tt{display:inline;max-width:initial;padding:0;margin:0;overflow:initial;line-height:inherit;word-wrap:normal;background-color:transparent;border:0}pre code:before,pre code:after{content:normal}pre tt:before,pre tt:after{content:normal}.poetry pre{font-family:Georgia, Garamond, serif !important;font-style:italic;font-size:110% !important;line-height:1.6em;display:block;margin-left:1em}.poetry pre code{font-family:Georgia, Garamond, serif !important;word-break:break-all;word-break:break-word;-webkit-hyphens:auto;-moz-hyphens:auto;hyphens:auto;white-space:pre-wrap}sup,sub,a.footnote{font-size:1.4ex;height:0;line-height:1;vertical-align:super;position:relative}sub{vertical-align:sub;top:-1px}@media print{body{background:#fff}img,table,figure{page-break-inside:avoid}#wrapper{background:#fff;border:none !important;font-size:12px}pre code{overflow:visible}}@media screen{body.inverted{border-color:#555;box-shadow:none;color:#eee !important}.inverted #wrapper,.inverted hr,.inverted p,.inverted td,.inverted li,.inverted h1,.inverted h2,.inverted h3,.inverted h4,.inverted h5,.inverted h6,.inverted th,.inverted .math,.inverted caption,.inverted dd,.inverted dt,.inverted blockquote{border-color:#555;box-shadow:none;color:#eee !important}.inverted td,.inverted th{background:#333}.inverted pre,.inverted code,.inverted tt{background:#eeeeee !important;color:#111}.inverted h2{border-color:#555555}.inverted hr{border-color:#777;border-width:1px !important}::selection{background:rgba(157,193,200,0.5)}h1::selection{background-color:rgba(45,156,208,0.3)}h2::selection{background-color:rgba(90,182,224,0.3)}h3::selection,h4::selection,h5::selection,h6::selection,li::selection,ol::selection{background-color:rgba(133,201,232,0.3)}code::selection{background-color:rgba(0,0,0,0.7);color:#eeeeee}code span::selection{background-color:rgba(0,0,0,0.7) !important;color:#eeeeee !important}a::selection{background-color:rgba(255,230,102,0.2)}.inverted a::selection{background-color:rgba(255,230,102,0.6)}td::selection,th::selection,caption::selection{background-color:rgba(180,237,95,0.5)}.inverted{background:#0b2531;background:#252a2a}.inverted #wrapper{background:#252a2a}.inverted a{color:#acd1d5}}.highlight{background:#fff}.highlight .c{color:#998;font-style:italic}.highlight .err{color:#a61717;background-color:#e3d2d2}.highlight .k,.highlight .o{font-weight:700}.highlight .cm{color:#998;font-style:italic}.highlight .cp{color:#999;font-weight:700}.highlight .c1{color:#998;font-style:italic}.highlight .cs{color:#999;font-weight:700;font-style:italic}.highlight .gd{color:#000;background-color:#fdd}.highlight .gd .x{color:#000;background-color:#faa}.highlight .ge{font-style:italic}.highlight .gr{color:#a00}.highlight .gh{color:#999}.highlight .gi{color:#000;background-color:#dfd}.highlight .gi .x{color:#000;background-color:#afa}.highlight .go{color:#888}.highlight .gp{color:#555}.highlight .gs{font-weight:700}.highlight .gu{color:purple;font-weight:700}.highlight .gt{color:#a00}.highlight .kc,.highlight .kd,.highlight .kn,.highlight .kp,.highlight .kr{font-weight:700}.highlight .kt{color:#458;font-weight:700}.highlight .m{color:#099}.highlight .s{color:#d14}.highlight .n{color:#333}.highlight .na{color:teal}.highlight .nb{color:#0086b3}.highlight .nc{color:#458;font-weight:700}.highlight .no{color:teal}.highlight .ni{color:purple}.highlight .ne,.highlight .nf{color:#900;font-weight:700}.highlight .nn{color:#555}.highlight .nt{color:navy}.highlight .nv{color:teal}.highlight .ow{font-weight:700}.highlight .w{color:#bbb}.highlight .mf,.highlight .mh,.highlight .mi,.highlight .mo{color:#099}.highlight .sb,.highlight .sc,.highlight .sd,.highlight .s2,.highlight .se,.highlight .sh,.highlight .si,.highlight .sx{color:#d14}.highlight .sr{color:#009926}.highlight .s1{color:#d14}.highlight .ss{color:#990073}.highlight .bp{color:#999}.highlight .vc,.highlight .vg,.highlight .vi{color:teal}.highlight .il{color:#099}.highlight .gc{color:#999;background-color:#EAF2F5}.type-csharp .highlight .k,.type-csharp .highlight .kt{color:blue}.type-csharp .highlight .nf{color:#000;font-weight:400}.type-csharp .highlight .nc{color:#2b91af}.type-csharp .highlight .nn{color:#000}.type-csharp .highlight .s,.type-csharp .highlight .sc{color:#a31515}.type-csharp .highlight .k,.type-csharp .highlight .kt{color:#00F}.type-csharp .highlight .nf{color:#000;font-weight:normal}.type-csharp .highlight .nc{color:#2B91AF}.type-csharp .highlight .nn{color:#000}.type-csharp .highlight .s,.type-csharp .highlight .sc{color:#A31515}body.dark #wrapper{background:transparent !important;box-shadow:none !important}kbd{background-color:#fafbfc;border:1px solid #d1d5da;border-radius:3px;box-shadow:inset 0 -1px 0 #d1d5da;color:#444d56;display:inline-block;font:11px SFMono-Regular, Consolas, Liberation Mono, Menlo, monospace;line-height:10px;padding:3px 5px;vertical-align:middle}.inverted kbd{background-color:#666;color:#fff}
#mkreplaced-toc{list-style-position:inside;padding:0;margin:0 0 0 1rem;list-style-type:none}#mkreplaced-toc li::before{content:''}#mkreplaced-toc li{font-size:1rem;line-height:1.25;font-weight:normal}#mkreplaced-toc li ul{font-size:1.3rem;font-weight:300;padding:.5rem 0;margin:0 0 0 1rem}#mkreplaced-toc li.missing{list-style-type:none !important}#mkreplaced-toc.max-1 ul,#mkreplaced-toc.max1 ul{display:none}#mkreplaced-toc.max-2 ul ul,#mkreplaced-toc.max2 ul ul{display:none}#mkreplaced-toc.max-3 ul ul ul,#mkreplaced-toc.max3 ul ul ul{display:none}#mkreplaced-toc.max-4 ul ul ul ul,#mkreplaced-toc.max4 ul ul ul ul{display:none}#mkreplaced-toc.max-5 ul ul ul ul ul,#mkreplaced-toc.max5 ul ul ul ul ul{display:none}.mk-rtl{direction:rtl;text-align:right}body.mkkatex-number-equations{counter-reset:eqnum}body.mkkatex-number-equations .katex-display{position:relative}body.mkkatex-number-equations .katex-display::after{counter-increment:eqnum;content:"(" counter(eqnum) ")";position:absolute;left:0;top:25%}body.mkkatex-number-equations.mkkatex-number-equations-right .katex-display::after{right:0;left:auto}.mkprinting,.mkprinting #wrapper{height:auto;margin-bottom:0;padding-bottom:0}.hideProgress #generated-toc,.hideProgress #firstdiff,.hideProgress #toc-title,.hideProgress #mkdocumentprogress,.hideProgress #mkincludechart,.hideProgress #mkprogressbar1,.hideProgress #mkprogressbar2,.hideProgress b.bookmark,.hideProgress .mkscrollmeter,.hideProgress #alllinks,.hideProgress #criticnav,.hideProgress .popup,.hideProgress #progressindicator,.hideProgress #mkautoscroll,.mkprinting #generated-toc,.mkprinting #firstdiff,.mkprinting #toc-title,.mkprinting #mkdocumentprogress,.mkprinting #mkincludechart,.mkprinting #mkprogressbar1,.mkprinting #mkprogressbar2,.mkprinting b.bookmark,.mkprinting .mkscrollmeter,.mkprinting #alllinks,.mkprinting #criticnav,.mkprinting .popup,.mkprinting #progressindicator,.mkprinting #mkautoscroll{display:none !important}.hideProgress .mkstyledtag,.mkprinting .mkstyledtag{display:none}.mkcolor-grammar-error,.mkcolor-spell-error{background:none;border-bottom:none}.mkprinting.mkshowcomments .mkstyledtag{display:inline;background:#ccc;padding:3px 9px;border-radius:20px;font-size:1}@media print{body{background:white;line-height:1.4}html,body,#wrapper{-moz-box-shadow:none;-webkit-box-shadow:none;box-shadow:none;-webkit-perspective:none !important;-webkit-text-size-adjust:none;border:0;box-sizing:border-box;float:none;margin:0;max-width:100%;padding:0;margin-top:0;width:auto}.critic #wrapper mark.crit{background-color:#fffd38 !important;text-decoration:none;color:#000}h1,h2,h3,h4,h5,h6{page-break-after:avoid}p,h2,h3{orphans:3;widows:3}section{page-break-before:avoid}pre>code{white-space:pre;word-break:break-word}#generated-toc,#firstdiff,#toc-title,#mkdocumentprogress,#mkincludechart,#mkprogressbar1,#mkprogressbar2,.mkscrollmeter,#alllinks,.popup{display:none !important}.suppressprintlinks a{border-bottom:none !important;color:inherit !important;cursor:default !important;text-decoration:none !important}.hrefafterlinktext #wrapper a:link:after,.hrefafterlinktext #wrapper a:visited:after{content:" (" attr(href) ") ";font-size:90%;opacity:.9}.nocodebreak pre{page-break-inside:avoid}img,table,figure{page-break-inside:avoid}.breakfootnotes .footnotes{page-break-before:always}.breakfootnotes .footnotes hr{display:none}#mktoctitle{display:block}#print-title{border-bottom:solid 1px #666;display:block}#wrapper pre{white-space:pre;white-space:pre-wrap;word-wrap:break-word}#wrapper #generated-toc-clone,#wrapper #mkreplaced-toc{display:block}.task-list{padding-left:3.3rem}.mkstyle--ink .task-list,.mkstyle--swiss .task-list{padding-left:3.3rem !important}.mkstyle--upstandingcitizen .task-list,.mkstyle--github .task-list{padding-left:3.6rem !important}.mkstyle--manuscript .task-list{padding-left:2.4rem !important}.mkstyle--amblin .task-list{padding-left:2.1rem !important}.mkstyle--grump .task-list{padding-left:1rem !important}.mkstyle--grump .task-list .task-list-item-checkbox{left:0 !important}.task-list .task-list-item{list-style-type:none !important;left:auto}.task-list .task-list-item .task-list-item-checkbox{-webkit-appearance:none;position:relative;left:auto}.task-list .task-list-item .task-list-item-checkbox:before{border:solid 1px #aaa;border-radius:2px;color:white;content:' ';display:block;font-weight:bold;height:1em;left:-1rem;line-height:1;position:absolute;text-align:center;top:-.75em;width:1em}.task-list .gh-complete.task-list-item .task-list-item-checkbox:before{background:#838387;content:'\2713'}}
#wrapper #generated-toc-clone,#wrapper #mkreplaced-toc,#wrapper #generated-toc-clone ul,#wrapper #mkreplaced-toc ul{list-style-position:inside}#wrapper #generated-toc-clone li.missing,#wrapper #mkreplaced-toc li.missing{list-style-type: none!important}#wrapper #generated-toc-clone ul,#wrapper #mkreplaced-toc ul{list-style-type: upper-roman}#wrapper #generated-toc-clone>ul>li>ul,#wrapper #mkreplaced-toc>li>ul {list-style-type: decimal}#wrapper #generated-toc-clone>ul>li>ul>li>ul,#wrapper #mkreplaced-toc>li>ul>li>ul{list-style-type: decimal-leading-zero}#wrapper #generated-toc-clone>ul>li>ul>li>ul>li>ul,#wrapper #mkreplaced-toc>li>ul>li>ul>li>ul{list-style-type: lower-greek}#wrapper #generated-toc-clone>ul>li>ul>li>ul>li>ul>li>ul,#wrapper #mkreplaced-toc>li>ul>li>ul>li>ul>li>ul{list-style-type: disc}#wrapper #generated-toc-clone>ul>li>ul>li>ul>li>ul>li>ul>li>ul,#wrapper #mkreplaced-toc>li>ul>li>ul>li>ul>li>ul>li>ul{list-style-type: square}#wrapper #generated-toc-clone,#wrapper #mkreplaced-toc{}
`
