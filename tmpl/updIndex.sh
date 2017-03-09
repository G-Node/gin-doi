sed "35i <tr>\
<td>\
<a href=\"{{.DoiInfo.UUID}}\">{{.DoiInfo.Title}}</a>\
</td>\
<td>2017-02-01</td>\
<td><a href=\"https://doi.org/{{.DoiInfo.DOI}}\" class =\"label label-default\">{{.DoiInfo.DOI}}</a></td>\
</tr>" ../index.html
