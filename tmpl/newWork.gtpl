<html>
<head>
	<title>Brucheion</title>
	<meta http-equiv="Content-Type" content="text/html; charset=UTF-8"/>
	<link rel="stylesheet" type="text/css" href="http://localhost{{.port}}/static/css/bootstrap.min.css">
	<link rel="stylesheet" type="text/css" href="http://localhost{{.port}}/static/css/bootstrap-theme.min.css">
	<link rel="stylesheet" type="text/css" href="http://localhost{{.port}}/static/css/application.css">
	<link rel="stylesheet" href="http://localhost{{.port}}/static/css/font-awesome.min.css">
	<link rel="stylesheet" type="text/css" href="http://localhost{{.port}}/static/css/bulma.css">
	<style>
        .tile-resizable {
            resize: both;
          overflow: hidden
        }
    </style>
</head>
<body>
<div class="container is-fluid">
	<nav class="nav has-shadow">
	  <div class="container">
	    <div class="nav-left">
	      <p class="nav-item">
	        <img src="/static/img/BrucheionLogo.png" alt="Brucheion logo">
	      </p>
	      <p class="nav-item is-tab is-hidden-mobile is-active">Brucheion 1.0.0</p>
	    </div>
			<div>
				<p class="button is-primary is-large is-inverted">EDIT CATALOG DATA</p>
			</div>
	    <div class="nav-right nav-menu">
	      <p class="nav-item is-tab">
	        User: {{printf "%s" .User}}
	      </p>
	    </div>
			<div class="nav-right nav-menu">
				<p class="nav-item is-tab">
					<a href="http://localhost{{.Port}}/{{.User}}/export/{{.User}}/"><i class="fa fa-download">CEX-Download</i></a>
				</p>
			</div>

	  </div>
	</nav>
	<section>
	<div class="tile is-ancestor">
			<div class="tile is-parent">
					<div class="tile is-child">
<form method="post" action="newWork">
								<div class="form-group">
										<label for="workurn">WorkURN:</label>
										<input type="text" class="form-control" name="workurn">
										<span class="help-block">Please enter a CTS URN: It has 4 colons (and ends on one ending on one). After "urn:cts:[yourcollection]:", you have to provide at least workgroup ID and work ID separated by a ".". Additionaly, you can provide version and exemplar IDs.</span>
										<label for="scheme">Scheme:</label>
										<input type="text" class="form-control" name="scheme">
										<span class="help-block">What is the citation scheme of the work? For example, "1.1.1" could resemble "book/chapter/paragraph".</span>
										<label for="workgroup">Workgroup:</label>
										<input type="text" class="form-control" name="workgroup">
										<span class="help-block">Workgroup in natural language.</span>
										<label for="title">Title:</label>
										<input type="text" class="form-control" name="title">
										<span class="help-block">Title in natural language.</span>
										<label for="version">Version:</label>
										<input type="text" class="form-control" name="version">
										<span class="help-block">Version in natural language.</span>
										<label for="exemplar">Exemplar:</label>
										<input type="text" class="form-control" name="exemplar">
										<span class="help-block">Exemplar in natural language.</span>
										<label for="online">Online:</label>
										<input type="text" class="form-control" name="online">
										<span class="help-block">Boolean; usually "true".</span>
										<label for="language">Language:</label>
										<input type="text" class="form-control" name="language">
										<span class="help-block">Language ID</span>
                                        <input class="button is-primary" type="submit" value="Save">
									</div>
                                    </div>
                                    </div>
                                    </div>
                                    </section>
</body>
</html>
