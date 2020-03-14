resource "aws_s3_bucket" "institutions" {
  bucket = "coronavirus.institutions.data"
  acl= "private"

  tags = {
    Name        = "coronavirus.institutions.data"
    Environment = "Dev"
  }
}