resource "aws_vpc_ipam" "default" {
  operating_regions {
    region_name = local.aws_region
  }

  tags = local.tags
}

resource "aws_vpc_ipam_pool" "default" {
  address_family = "ipv4"
  ipam_scope_id  = aws_vpc_ipam.default.private_default_scope_id
  locale         = local.aws_region

  tags = local.tags
}

resource "aws_vpc_ipam_pool_cidr" "default_private" {
  ipam_pool_id = aws_vpc_ipam_pool.default.id
  cidr         = "10.0.1.0/24"
}

resource "aws_vpc" "default" {
  ipv4_ipam_pool_id   = aws_vpc_ipam_pool.default.id
  ipv4_netmask_length = 28
  depends_on = [
    aws_vpc_ipam_pool_cidr.default_private
  ]

  tags = local.tags
}

resource "aws_subnet" "main" {
  vpc_id     = aws_vpc.default.id
  cidr_block = "10.0.1.0/31"

  tags = local.tags
}

resource "aws_security_group" "default" {
  name        = "AllowOnlyTcpInboundAndOpenOutboundForAll"
  description = "Allow only tcp inbound & no restriction to outbound"
  vpc_id      = aws_vpc.default.id

  tags = local.tags
}

resource "aws_vpc_security_group_ingress_rule" "allow_tcp" {
  security_group_id = aws_security_group.default.id
  cidr_ipv4         = aws_vpc.default.cidr_block
  from_port         = 443
  ip_protocol       = "tcp"
  to_port           = 443
}

# resource "aws_vpc_security_group_ingress_rule" "allow_tls_ipv6" {
#   security_group_id = aws_security_group.default.id
#   cidr_ipv6         = aws_vpc.default.ipv6_cidr_block
#   from_port         = 443
#   ip_protocol       = "tcp"
#   to_port           = 443
# }

resource "aws_vpc_security_group_egress_rule" "allow_all_traffic_ipv4" {
  security_group_id = aws_security_group.default.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1" # semantically equivalent to all ports
}

resource "aws_vpc_security_group_egress_rule" "allow_all_traffic_ipv6" {
  security_group_id = aws_security_group.default.id
  cidr_ipv6         = "::/0"
  ip_protocol       = "-1" # semantically equivalent to all ports
}
