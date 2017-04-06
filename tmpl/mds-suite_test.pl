#!/usr/bin/perl
use strict;
use feature qw(say);
use Switch;
use Getopt::Std;
use Pod::Usage;
use LWP;
use Crypt::SSLeay;    # SSL for LWP
use Term::ReadKey;    # for password reading
use URI;
use URI::Escape;

my $GLOBAL_SERVER     = '';
my $LOCAL_SERVER      = '';
my $DEFAULT_DC_SYMBOL = '';
my $DEFAULT_DC_PW     = '';
my $DEFAULT_AL_SYMBOL = '';
my $DEFAULT_AL_PW     = '';
my %opts;             #Getopt::Std

sub main() {
  getopts("c:hlnp:stu:v", \%opts) or pod2usage();
  pod2usage() if $opts{h};
  
  $ENV{PERL_LWP_SSL_VERIFY_HOSTNAME} = 0 if $opts{l};

  my ($resource, $method, %query, $content, $content_type);
  my $default_user_name = $DEFAULT_DC_SYMBOL;
  my $default_user_pw   = $DEFAULT_DC_PW;
  my $command = lc shift @ARGV or pod2usage("missing command");
  switch ($command) {
    case "metadata" {
      $resource = 'metadata';
      $content_type = 'application/xml;charset=UTF-8';
      $method = uc shift @ARGV or pod2usage("missing method");
      if ($method =~ "GET|DELETE|PUT") {
        my $doi = shift @ARGV or pod2usage("missing doi");
        $resource .= "/" . escape($doi);
      }
    }
    case "media" {
      $resource = 'media';
      $content_type = 'text/plain;charset=UTF-8';
      $method = uc shift @ARGV or pod2usage("missing method");
      my $doi = shift @ARGV or pod2usage("missing doi");
      $resource .= "/" . escape($doi);
    }
    case "doi" {
      $resource = 'doi';
      $content_type = 'text/plain;charset=UTF-8';
      $method = uc shift @ARGV or pod2usage("missing method");
      my $doi = shift @ARGV or pod2usage("missing doi (or '-')");
      if ($doi ne "-") {
        if ($method =~ "GET|PUT") {
          $resource .= "/" . escape($doi);
        }
        if ($method =~ "PUT|POST") {
          my $url = shift @ARGV or pod2usage("missing url");
          if ($url ne "-") {
            $content = "doi=$doi\nurl=$url";
          }
        }
      }
    }
    case "datacentre" {
      $resource = 'datacentre';
      $default_user_name = $DEFAULT_AL_SYMBOL;
      $default_user_pw   = $DEFAULT_AL_PW;
      $content_type = 'application/xml;charset=UTF-8';
      $method = uc shift @ARGV or pod2usage("missing method");
      my $symbol = shift @ARGV or pod2usage("missing symbol");
      $query{symbol} = $symbol;
    }
    case "generic" {
      $method = uc shift @ARGV or pod2usage("missing method");
      $resource = shift @ARGV or pod2usage("missing resource");
      $content_type = $opts{c};
    }  
    else { pod2usage("unknown command '$command'"); }
  }
  
  if (!$content and $method =~ "POST|PUT") {
      my @content = <>;
      $content = "@content";
      chomp $content;
  }
  
  my $user_name = $opts{u} || $default_user_name;
  my $user_pw = $opts{p} || ($opts{u} ? read_pw() : $default_user_pw);

  my $domain = $opts{l} ? $LOCAL_SERVER : $GLOBAL_SERVER;
  
  $query{testMode} = "true" if $opts{t};
  
  my $url = URI->new("https://$domain/$resource");
  $url->query_form(%query);

  my $response_code =  do_request($method, $url,
    $user_name, $user_pw, $content, $content_type);
    
  exit $response_code;
}

sub escape {
  my $str = shift;
  return uri_escape($str, "#?");
}

sub read_pw {
  print STDERR "password: ";
  ReadMode('noecho');
  my $pw = ReadLine(0);
  chomp $pw;
  ReadMode('restore');
  return $pw;
}

sub do_request {
  my ($method, $url, $user_name, $user_pw, $content, $content_type) = @_;

  # build request
  my $headers = HTTP::Headers->new(
    Accept         => 'application/xml',
    'Content-Type' => $content_type
  );
  my $req = HTTP::Request->new(
    $method => $url,
    $headers, $content
  );
  $req->authorization_basic($user_name, $user_pw) unless $opts{n};

  # pass request to the user agent and get a response back
  my $ua = LWP::UserAgent->new;
  my $res = $ua->request($req);

  # output request/response
  if ($opts{v}) {
    say STDERR "== REQUEST ==";
    say STDERR $req->method . " " . $req->uri->as_string();
    say STDERR $req->headers_as_string();
    say STDERR shorten($req->content) if $req->content;
    say STDERR "\n== RESPONSE ==";
  }
  say STDERR $res->status_line;
  say STDERR $res->headers_as_string() if $opts{v};
  say shorten($res->content) if $res->content;
  
  return $res->code();
}


sub shorten {
  my ($txt) = @_;
  return $txt unless $opts{s};
  my @rows = split "\n", $txt;
  my $rows = scalar(@rows);
  my $size = length($txt);
  return "+++ body: $rows rows, $size chars; first 60 characters:\n+++ " . substr("@rows",0,60);
}

main();

__END__

=head1 NAME

 mds-suite

=head1 SYNOPSIS

 mds-suite [options] <command> 

 Options:
   -c <type>   - set content-type header (only for command 'generic') 
   -h          - prints this help
   -l          - use a local test server
   -n          - no credentials (only for testing)
   -s          - short output (truncate request/response body)
   -t          - enable testMode
   -u <symbol> - username (defaults to value specified in the script)
   -v          - verbose (display complete request and response)

 Commands:
   datacentre <method> <symbol>
   doi <method> (<doi> <url> | '-')
   media <GET|POST> <doi> 
   metadata <POST|PUT>
   metadata <DELETE|GET> <doi>
 
   [ generic <method> <resource/params> ]
 
 The body of an http POST/PUT request is read from stdin. 
 For 'doi put/post' the request body is build from commandline params,
 unless you set '-' (=read from stdin) as doi param.  
