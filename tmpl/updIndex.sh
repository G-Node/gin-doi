sed "35i <tr>\
<td>\
<a href=\"{{.DOIInfo.UUID}}\">{{.DOIInfo.Title}}</a>\
</td>\
<td>2017-02-01</td>\
<td><a href=\"https://doi.org/{{.DOIInfo.DOI}}\" class =\"label label-default\">{{.DOIInfo.DOI}}</a></td>\
</tr>" ../index.html
