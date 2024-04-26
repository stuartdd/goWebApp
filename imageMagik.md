datetime=`magick sky.jpg -format "%[EXIF:datetime]\n" info: | tr -d ":" | tr " " "-"`
echo "$datetime"

