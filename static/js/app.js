User = Backbone.Model.extend({
    urlRoot: '/api/user/',
    defaults: {
        id: null,
        email: null,
        firstname: null,
        secondname: null,
        favorites: [],
        blacklist: [],
        phone: null,
        token: null
    },
    initialize: function () {
        console.log("hello");
        this.set({urlRoot: '/api/user/' + this.get('id')});
        this.set({url: '/user/' + this.get('id')})
    }
});

var user;
var Users = Backbone.Collection.extend({
    model: User,
    url: '/api/user/'
});

var users = new Users;

var token, userId;

token = $.cookie('token');
userId = $.cookie('userId');

var connection;

function hideLogin() {
    console.log(user.get('firstname'));
    var template = _.template($('#form-user-template').html(), {user: user});
    $("#userModal").html(template);
//    template = _.template($('#block-user-template').html(), {user: user});
//    var block_user = $('#content-wrapper');
//    block_user.html(template);
    $('#block-login').hide();
    $('#button-logout').show();
//    block_user.show();
}

function logout() {
    $.removeCookie('token');
    $.removeCookie('userId');
    $('#button-logout').hide();
    $('#block-user').hide();
    token = null;
    user = null;
    userId = null;
}

$('#button-logout').click(function (e) {
    e.preventDefault();
    logout();
    $('#block-login').show();
});
function setup_user_form() {
    var form = document.getElementById('form-user');
    $('#form-user-save').on('click', function(e){
        e.preventDefault();
        var data = $(form).serialize();
        console.log(data);
    });

    $('#form-user-file').on('change', function (e) {
        e.preventDefault();
        var bar = $('#form-user-file-progress');
        bar.show();
        bar.find('.progress-bar').attr('aria-valuenow', 0);
        bar.find('.progress-bar').css('width', 0 + '%');
        console.log(form);
        var data = new FormData(form);
        console.log(data);
        $.ajax({
            url: '/api/image',
            type: 'POST',
            data: data,
            processData: false, // Don't process the files
            contentType: false, // Set content type to false as jQuery will tell the server its a query string request
            success: function (data, textStatus, jqXHR) {
                if (typeof data.error === 'undefined') {
                    // Success so call function to process the form
                    console.log(data);
                    $('#form-user-image').attr('src', data.url);
                    $('#form-user-image-hidden').attr('value', JSON.stringify(data));
                    user.set('photo', data);
                    user.save();
                    bar.find('.progress-bar').css('width', 100 + '%');
                    setTimeout(function () {
                        bar.hide()
                    }, 500);
                }
                else {
                    // Handle errors here
                    console.log('ERRORS: ' + data.error);
                }
            },
            error: function (jqXHR, textStatus, errorThrown) {
                // Handle errors here
                console.log('ERRORS: ' + textStatus);
                // STOP LOADING SPINNER
            }
        });
    });
}
$('#form-video-file').on('change', function (e) {
    e.preventDefault();
    var form = document.getElementById('form-video');
    var bar = $('#form-video-file-progress');
    bar.show();
    bar.find('.progress-bar').attr('aria-valuenow', 0);
    bar.find('.progress-bar').css('width', 50 + '%');
    console.log(form);
    var data = new FormData(form);
    console.log(data);
    $.ajax({
        url: '/api/image',
        type: 'POST',
        data: data,
        processData: false, // Don't process the files
        contentType: false, // Set content type to false as jQuery will tell the server its a query string request
        success: function (data, textStatus, jqXHR) {
            if (typeof data.error === 'undefined') {
                // Success so call function to process the form
                console.log(data);
                $('#form-video-src').attr('src', data.url);
                bar.find('.progress-bar').css('width', 100 + '%');
                setTimeout(function () {
                    bar.hide()
                }, 500);
                var v = document.getElementById('form-video-video');
                v.load();
                v.play();
            }
            else {
                // Handle errors here
                console.log('ERRORS: ' + data.error);
            }
        },
        error: function (jqXHR, textStatus, errorThrown) {
            // Handle errors here
            console.log('ERRORS: ' + textStatus);
            // STOP LOADING SPINNER
        }
    });
});

var userInfo;
var userMain;
function connect() {
    hideLogin();
    setup_user_form();
    userInfo = new UserInfo({model:user, el: $("#user-info")});
    userMain = new UserMain({model:user});
    userMain.render();
    connection = new WebSocket('ws://poputchiki.ru/api/realtime');
    connection.onopen = function () {
        console.log('ws connected')
    };

    connection.onmessage = function (event) {
        data = JSON.parse(event.data);
        console.log("got message", data);

        if (data.type == 'progressmessage') {
            console.log('progress ', data.body.progress);
            var bar = $('#form-user-file-progress').find('.progress-bar');
            bar.attr('aria-valuenow', data.body.progress);
            bar.css('width', data.body.progress + '%')
        }
    };

    connection.onclose = function () {
        console.log('ws closed')
    }
}

if (token != null && userId != null) {
    console.log(token, userId);
    user = new User({id: userId});
    user.fetch({success: function () {
        connect();
    }});
    console.log("auth ok")
}

function login(data) {
    token = data.token;
    userId = data.id;
    $.cookie('token', token);
    $.cookie('userId', userId);
    console.log(token, userId);
    console.log($.cookie('token'));
    user = new User({id: userId});
    user.fetch({success: function () {
        connect();
    }});
}

$("#form-register-submit").click(function (e) {
    e.preventDefault();
    var url = "/api/auth/register"; // the script where you handle the form input.

    $.ajax({
        type: "POST",
        url: url,
        data: $("#form-register").serialize(), // serializes the form's elements.
        success: function (data) {
            login(data);
            $('#myModal').modal('hide');
        }
    });
});
$("#form-login-submit").click(function (e) {
    e.preventDefault();
    var url = "/api/auth/login"; // the script where you handle the form input.

    $.ajax({
        type: "POST",
        url: url,
        data: $("#form-login").serialize(), // serializes the form's elements.
        success: function (data) {
            login(data);
            $('#loginModal').modal('hide');
        }
    });
});


var UserRouter = Backbone.Router.extend({
    routes: {
        "user/:id": "userPage",
        "": "mainPage"
    },

    userPage: function (id) {
        $("#content-wrapper").html(userMain.el)
    },

    mainPage: function() {
        if (userMain) {
            $("#content-wrapper").html(userMain.el)
        } else {

        }
    }
});

var userRouter = new UserRouter();


var UserInfo = Backbone.View.extend({
  template: _.template($('#user-info-template').html()),
  className: "user-info",

  initialize: function() {
    console.log('init');
    _.bindAll(this, 'render');
    this.render()
  },

  render: function() {
    $(this.el).html(this.template({user: this.model}));
  }

});

var UserMain = Backbone.View.extend({
  template: _.template($('#block-user-template').html()),
  className: "user-info",

  initialize: function() {
    _.bindAll(this, 'render');
  },

  render: function() {
    $(this.el).html(this.template({user: this.model}));
  }

});

var UserBlock = Backbone.View.extend({
  template: _.template($('#block-user-info-template').html()),
  className: "user-info",

  initialize: function() {
    _.bindAll(this, 'render');
  },

  render: function() {
    $(this.el).html(this.template({user: this.model}));
  }

});



Backbone.history.start({pushState: true});
$(document).on("click", "a:not([data-bypass])", function(evt) {
    // Get the anchor href and protcol
    var href = $(this).attr("href");
    var protocol = this.protocol + "//";
    // Ensure the protocol is not part of URL, meaning its relative.
    if (href && href.slice(0, protocol.length) !== protocol &&
        href.indexOf("javascript:") !== 0) {
      // Stop the default event to ensure the link will not cause a page
      // refresh.
      evt.preventDefault();

      // `Backbone.history.navigate` is sufficient for all Routers and will
      // trigger the correct events.  The Router's internal `navigate` method
      // calls this anyways.
      Backbone.history.navigate(href, true);
    }
 });